package agent

import (
	"regexp"
	"time"

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"

	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	corelistersv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/containership/cluster-manager/pkg/constants"
	"github.com/containership/cluster-manager/pkg/env"
	"github.com/containership/cluster-manager/pkg/geohash"
	"github.com/containership/cluster-manager/pkg/log"
)

const (
	nodeLabelControllerName = "NodeLabelController"
)

const (
	// TODO update this when the label graduates from beta
	failureDomainRegionLabelKey = "failure-domain.beta.kubernetes.io/region"
)

const (
	maxRetriesNodeLabelController = 5
)

// NodeLabelController reconciles node labels
type NodeLabelController struct {
	clientset kubernetes.Interface

	nodeLister  corelistersv1.NodeLister
	nodesSynced cache.InformerSynced

	workqueue workqueue.RateLimitingInterface
}

// NewNodeLabelController creates a new agent NodeLabelController
func NewNodeLabelController(
	clientset kubernetes.Interface,
	informerFactory informers.SharedInformerFactory) *NodeLabelController {

	c := &NodeLabelController{
		clientset: clientset,
		workqueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Nodes"),
	}

	nodeInformer := informerFactory.Core().V1().Nodes()

	nodeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueueNode,
		UpdateFunc: func(old, new interface{}) {
			oldNode := old.(*corev1.Node)
			newNode := new.(*corev1.Node)
			if oldNode.ResourceVersion == newNode.ResourceVersion {
				// This must be a syncInterval update - nothing has changed so
				// do nothing
				return
			}
			c.enqueueNode(new)
		},
		// We don't care about deletes
	})

	c.nodeLister = nodeInformer.Lister()
	c.nodesSynced = nodeInformer.Informer().HasSynced

	return c
}

// Run kicks off the NodeLabelController with the given number of workers to process the
// workqueue
func (c *NodeLabelController) Run(numWorkers int, stopCh chan struct{}) error {
	defer runtime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	log.Infof("Starting %s", nodeLabelControllerName)

	log.Infof("%s: Waiting for informer caches to sync", nodeLabelControllerName)
	if ok := cache.WaitForCacheSync(stopCh, c.nodesSynced); !ok {
		// If this routine is unable to wait for the caches to sync then we
		// stop everything
		close(stopCh)
		return errors.New("failed to wait for caches to sync")
	}

	log.Infof("%s: Starting workers", nodeLabelControllerName)
	// Launch two workers to process Node resources
	for i := 0; i < numWorkers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	log.Infof("%s: Started workers", nodeLabelControllerName)
	<-stopCh
	log.Info("%s: Shutting down controller", nodeLabelControllerName)

	return nil
}

// runWorker continually requests that the next queue item be processed
func (c *NodeLabelController) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem continually pops items off of the workqueue and handles
// them
func (c *NodeLabelController) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workqueue.Forget(obj)
			log.Errorf("expected string in workqueue but got %#v", obj)
			return nil
		}

		// Run the syncHandler, passing it the namespace/name string of the
		// Node resource to be synced.
		err := c.syncHandler(key)
		return c.handleErr(err, key)
	}(obj)

	if err != nil {
		log.Error(err)
		return true
	}

	return true
}

// handleErr looks to see if the resource sync event returned with an error,
// if it did the resource gets requeued up to as many times as is set for
// the max retries. If retry count is hit, or the resource is synced successfully
// the resource is moved of the queue
func (c *NodeLabelController) handleErr(err error, key interface{}) error {
	if err == nil {
		c.workqueue.Forget(key)
		return nil
	}

	if c.workqueue.NumRequeues(key) < maxRetriesNodeLabelController {
		c.workqueue.AddRateLimited(key)
		return errors.Wrapf(err, "syncing key %s (has been resynced %d times)",
			key, c.workqueue.NumRequeues(key))
	}

	c.workqueue.Forget(key)
	log.Infof("Dropping %v out of the queue: %v", key, err)
	return err
}

func (c *NodeLabelController) enqueueNode(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		log.Error(err)
		return
	}
	c.workqueue.AddRateLimited(key)
}

func (c *NodeLabelController) syncHandler(key string) error {
	_, name, _ := cache.SplitMetaNamespaceKey(key)

	if name != env.NodeName() {
		// We only care about the node this agent is running on
		return nil
	}

	node, err := c.nodeLister.Get(name)
	if err != nil {
		if kubeerrors.IsNotFound(err) {
			// Node is no longer around so nothing to do
			return nil
		}

		return errors.Wrapf(err, "getting node %s for node label reconciliation", name)
	}

	provider := providerNameFromProviderID(node.Spec.ProviderID)
	if provider == "" {
		// Retrying won't help here
		log.Infof("%s: Skipping node %s because provider is not available",
			nodeLabelControllerName, node.Name)
		return nil
	}

	labels := node.Labels
	region, ok := labels[failureDomainRegionLabelKey]
	if !ok {
		// Retrying won't help here
		log.Infof("%s: Skipping node %s because region is not available",
			nodeLabelControllerName, node.Name)
		return nil
	}

	geohash, err := geohash.ForProviderAndRegion(provider, region)
	if err != nil {
		// Retrying here probably won't help here either, but return an error
		// for visibility. If we have a provider and region but our lookup
		// fails, then we probably need to update our lookup table. It'll be
		// dropped from the queue after a max number of retries anyway.
		return errors.Wrapf(err, "looking up geohash for provider %s, region %s",
			provider, region)
	}

	needsUpdate := false
	existing, ok := labels[constants.ContainershipNodeGeohashLabelKey]
	if !ok {
		needsUpdate = true
		log.Infof("Adding geohash label for node %s", node.Name)
	} else if existing != geohash {
		needsUpdate = true
		log.Infof("Fixing geohash label for node %s", node.Name)
	}

	if needsUpdate {
		labels[constants.ContainershipNodeGeohashLabelKey] = geohash

		nodeCopy := node.DeepCopy()
		nodeCopy.Labels = labels

		_, err = c.clientset.CoreV1().Nodes().Update(nodeCopy)
		if err != nil {
			return errors.Wrap(err, "updating node with new labels")
		}
	}

	return nil
}

// Given a provider ID in the format <ProviderName>://<ProviderSpecificNodeID>,
// extract the provider name. If the provider name cannot be extracted,
// an empty string is returned.
func providerNameFromProviderID(providerID string) string {
	r := regexp.MustCompile("^(.*)://")
	res := r.FindStringSubmatch(providerID)
	if len(res) != 2 {
		// This must be a malformed ID. Just return an empty string and make
		// the caller handle it.
		return ""
	}

	// Index 0 is the entire matched string, 1 is the first submatch
	return res[1]
}
