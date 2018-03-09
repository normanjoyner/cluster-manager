package coordinator

import (
	"fmt"
	"time"

	"github.com/containership/cloud-agent/internal/constants"
	"github.com/containership/cloud-agent/internal/log"
	"github.com/containership/cloud-agent/internal/tools"

	containershipv3 "github.com/containership/cloud-agent/pkg/apis/containership.io/v3"
	csclientset "github.com/containership/cloud-agent/pkg/client/clientset/versioned"
	csinformers "github.com/containership/cloud-agent/pkg/client/informers/externalversions"
	cslisters "github.com/containership/cloud-agent/pkg/client/listers/containership.io/v3"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"

	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	corelistersv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	corev1 "k8s.io/api/core/v1"
)

const (
	// UpgradeControllerName is the name of the controller being ran
	UpgradeControllerName = "upgrade"

	upgradeDelayBetweenRetries = 30 * time.Second

	maxUpgradeControllerRetries = 10
)

// UpgradeController is the controller implementation for the containership
// upgrading clusters to a users desired kubernetes version
type UpgradeController struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	csclientset   csclientset.Interface

	upgradeLister  cslisters.ClusterUpgradeLister
	upgradesSynced cache.InformerSynced
	nodeLister     corelistersv1.NodeLister
	nodesSynced    cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder
}

// NewUpgradeController returns a new upgrade controller
func NewUpgradeController(kubeclientset kubernetes.Interface, clientset csclientset.Interface, kubeInformerFactory kubeinformers.SharedInformerFactory, csInformerFactory csinformers.SharedInformerFactory) *UpgradeController {
	rateLimiter := workqueue.NewItemExponentialFailureRateLimiter(upgradeDelayBetweenRetries, upgradeDelayBetweenRetries)

	uc := &UpgradeController{
		kubeclientset: kubeclientset,
		csclientset:   clientset,
		workqueue:     workqueue.NewNamedRateLimitingQueue(rateLimiter, "Upgrade"),
		recorder:      tools.CreateAndStartRecorder(kubeclientset, UpgradeControllerName),
	}

	// Instantiate resource informers
	upgradeInformer := csInformerFactory.Containership().V3().ClusterUpgrades()
	nodeInformer := kubeInformerFactory.Core().V1().Nodes()

	// informers listens for add events an queues the cluster upgrade
	upgradeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: uc.enqueueUpgrade,
	})
	// TODO:// trigger a currentNode update when node updates annotation to finished
	// nodeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
	//     UpdateFunc: uc.enqueueNode,
	// })

	// Listers are used for cache inspection and Synced functions
	// are used to wait for cache synchronization
	uc.upgradeLister = upgradeInformer.Lister()
	uc.upgradesSynced = upgradeInformer.Informer().HasSynced
	uc.nodeLister = nodeInformer.Lister()
	uc.nodesSynced = nodeInformer.Informer().HasSynced

	log.Info(UpgradeControllerName, ": Setting up event handlers")

	return uc
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (uc *UpgradeController) Run(numWorkers int, stopCh chan struct{}) {
	defer runtime.HandleCrash()
	defer uc.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	log.Info(UpgradeControllerName, ": Starting controller")

	if ok := cache.WaitForCacheSync(
		stopCh,
		uc.upgradesSynced); !ok {
		// If this channel is unable to wait for caches to sync we stop both
		// the containership controller, and the upgrade controller
		close(stopCh)
		log.Error("failed to wait for caches to sync")
	}

	log.Info(UpgradeControllerName, ": Starting workers")
	// Launch numWorkers amount of workers to process resources
	for i := 0; i < numWorkers; i++ {
		go wait.Until(uc.runWorker, time.Second, stopCh)
	}

	log.Info(UpgradeControllerName, ": Started workers")
	<-stopCh
	log.Info(UpgradeControllerName, ": Shutting down workers")
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (uc *UpgradeController) runWorker() {
	for uc.processNextWorkItem() {
	}
}

// processNextWorkItem continually pops items off of the workqueue and handles
// them
func (uc *UpgradeController) processNextWorkItem() bool {
	obj, shutdown := uc.workqueue.Get()

	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer uc.workqueue.Done(obj)
		var key string
		var ok bool
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			uc.workqueue.Forget(obj)
			log.Errorf("expected string in workqueue but got %#v", obj)
			return nil
		}

		// Run the syncHandler, passing it the namespace/name string of the
		// Upgrade resource to be synced.
		err := uc.upgradeSyncHandler(key)
		return uc.handleErr(err, key)
	}(obj)

	if err != nil {
		log.Error(err)
		return true
	}

	return true
}

func (uc *UpgradeController) handleErr(err error, key interface{}) error {
	if err == nil {
		log.Debugf("Successfully synced '%s'", key)
		uc.workqueue.Forget(key)
		return nil
	}

	if uc.workqueue.NumRequeues(key) < maxUpgradeControllerRetries {
		uc.workqueue.AddRateLimited(key)
		return fmt.Errorf("error syncing %q: %s. Has been resynced %v times", key, err.Error(), uc.workqueue.NumRequeues(key))
	}

	uc.workqueue.Forget(key)
	log.Infof("Dropping Upgrade %q out of the queue: %v", key, err)
	return err
}

// upgradeSyncHandler looks at the UpgradeCluster resource and if it is in a
// completed state return. Otherwise, it finds the next node that should be
// updated and sets that as the current node on the the cluster resource.
func (uc *UpgradeController) upgradeSyncHandler(key string) error {
	_, name, _ := cache.SplitMetaNamespaceKey(key)
	upgrade, err := uc.upgradeLister.ClusterUpgrades(constants.ContainershipNamespace).Get(name)
	// If upgrade has already been fully processed and either Successed or Failed
	// we don't need to do anything
	if isDone(upgrade) {
		return nil
	}

	// if cluster is not in upgraded state
	node := uc.getNextNode(upgrade)
	if node == nil {
		return nil
	}

	currentUpgrade := upgrade.DeepCopy()
	currentUpgrade.Spec.CurrentNode = node.Name
	currentUpgrade.Spec.Status = containershipv3.UpgradeInProgress
	_, err = uc.csclientset.ContainershipV3().ClusterUpgrades(constants.ContainershipNamespace).Update(currentUpgrade)

	return err
}

// enqueueUpgrade enqueues the upgrade on add
func (uc *UpgradeController) enqueueUpgrade(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Error(err)
		return
	}

	uc.workqueue.AddRateLimited(key)
}

// isDone returns true if an upgrade has already been fully processed and has
// the status of either Successed or Failed
func isDone(cup *containershipv3.ClusterUpgrade) bool {
	if cup.Spec.Status == containershipv3.UpgradeSuccess || cup.Spec.Status == containershipv3.UpgradeFailed {
		return true
	}

	return false
}

// isCurrentNode checks to see if the node being looked at is the current node being processed
func isCurrentNode(cup *containershipv3.ClusterUpgrade, node *corev1.Node) bool {
	return cup.Spec.CurrentNode == node.Name
}

// getNextNode finds the next node to start updating
func (uc *UpgradeController) getNextNode(cup *containershipv3.ClusterUpgrade) *corev1.Node {
	masters, _ := uc.nodeLister.List(getMasterSelector(cup.Spec.LabelSelector))
	for _, master := range masters {
		if isNext(cup, master) {
			return master
		}
	}

	workers, _ := uc.nodeLister.List(getWorkerSelector(cup.Spec.LabelSelector))
	for _, worker := range workers {
		if isNext(cup, worker) {
			return worker
		}
	}

	return nil
}

func isNext(cup *containershipv3.ClusterUpgrade, node *corev1.Node) bool {
	return !tools.NodeIsTargetKubernetesVersion(cup, node) && !isCurrentNode(cup, node)
}

func addCustomLabelSelectors(selector labels.Selector, lss []containershipv3.LabelSelectorSpec) labels.Selector {
	for _, ls := range lss {
		nr, _ := labels.NewRequirement(ls.Label, ls.Operator, ls.Value)
		selector = selector.Add(*nr)
	}

	return selector
}

func getMasterSelector(lss []containershipv3.LabelSelectorSpec) labels.Selector {
	masterSelector := labels.Set(
		map[string]string{
			"node-role.kubernetes.io/master": "true",
		}).AsSelector()

	masterSelector = addCustomLabelSelectors(masterSelector, lss)
	return masterSelector
}

func getWorkerSelector(lss []containershipv3.LabelSelectorSpec) labels.Selector {
	masterLabelDNE, _ := labels.NewRequirement("node-role.kubernetes.io/master", selection.DoesNotExist, []string{})
	workerSelector := labels.NewSelector()
	workerSelector = workerSelector.Add(*masterLabelDNE)

	workerSelector = addCustomLabelSelectors(workerSelector, lss)
	return workerSelector
}
