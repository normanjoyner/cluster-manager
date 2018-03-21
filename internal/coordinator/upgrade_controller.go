package coordinator

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/api/errors"
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

	"github.com/containership/cloud-agent/internal/constants"
	"github.com/containership/cloud-agent/internal/log"
	"github.com/containership/cloud-agent/internal/tools"

	provisioncsv3 "github.com/containership/cloud-agent/pkg/apis/provision.containership.io/v3"
	csclientset "github.com/containership/cloud-agent/pkg/client/clientset/versioned"
	csinformers "github.com/containership/cloud-agent/pkg/client/informers/externalversions"
	pcslisters "github.com/containership/cloud-agent/pkg/client/listers/provision.containership.io/v3"
)

const (
	upgradeControllerName = "upgrade"

	upgradeDelayBetweenRetries = 30 * time.Second

	maxUpgradeControllerRetries = 10
)

// UpgradeController is the controller implementation for the containership
// upgrading clusters to a users desired kubernetes version
type UpgradeController struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	csclientset   csclientset.Interface

	upgradeLister  pcslisters.ClusterUpgradeLister
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

	// TODO we shouldn't be keeping state in RAM for upgrades
	currentUpgradeName string
}

// NewUpgradeController returns a new upgrade controller
func NewUpgradeController(kubeclientset kubernetes.Interface, clientset csclientset.Interface, kubeInformerFactory kubeinformers.SharedInformerFactory, csInformerFactory csinformers.SharedInformerFactory) *UpgradeController {
	rateLimiter := workqueue.NewItemExponentialFailureRateLimiter(upgradeDelayBetweenRetries, upgradeDelayBetweenRetries)

	uc := &UpgradeController{
		kubeclientset: kubeclientset,
		csclientset:   clientset,
		workqueue:     workqueue.NewNamedRateLimitingQueue(rateLimiter, "Upgrade"),
		recorder:      tools.CreateAndStartRecorder(kubeclientset, upgradeControllerName),
	}

	// Instantiate resource informers
	upgradeInformer := csInformerFactory.ContainershipProvision().V3().ClusterUpgrades()
	nodeInformer := kubeInformerFactory.Core().V1().Nodes()

	log.Info(upgradeControllerName, ": Setting up event handlers")

	upgradeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: uc.enqueueUpgrade,
	})

	nodeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(old, new interface{}) {
			oldNode := old.(*corev1.Node)
			newNode := new.(*corev1.Node)
			if oldNode.ResourceVersion == newNode.ResourceVersion {
				return
			}
			uc.enqueueNode(new)
		},
	})

	// Listers are used for cache inspection and Synced functions
	// are used to wait for cache synchronization
	uc.upgradeLister = upgradeInformer.Lister()
	uc.upgradesSynced = upgradeInformer.Informer().HasSynced
	uc.nodeLister = nodeInformer.Lister()
	uc.nodesSynced = nodeInformer.Informer().HasSynced

	log.Info(upgradeControllerName, ": Setting up event handlers")

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
	log.Info(upgradeControllerName, ": Starting controller")

	if ok := cache.WaitForCacheSync(
		stopCh,
		uc.upgradesSynced); !ok {
		// If this channel is unable to wait for caches to sync we stop both
		// the containership controller, and the upgrade controller
		close(stopCh)
		log.Error("failed to wait for caches to sync")
	}

	log.Info(upgradeControllerName, ": Starting workers")
	// Launch numWorkers amount of workers to process resources
	for i := 0; i < numWorkers; i++ {
		go wait.Until(uc.runWorker, time.Second, stopCh)
	}

	log.Info(upgradeControllerName, ": Started workers")
	<-stopCh
	log.Info(upgradeControllerName, ": Shutting down workers")
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

		kind, _, _, err := tools.SplitMetaResourceNamespaceKeyFunc(key)
		if err != nil {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			uc.workqueue.Forget(obj)
			log.Errorf("key is in incorrect format to process %#v", obj)
			return nil
		}

		// Run the needed sync handler, passing it the kind from the key string
		switch kind {
		case "upgrade":
			err := uc.upgradeSyncHandler(key)
			return uc.handleErr(err, key)
		case "node":
			err := uc.nodeSyncHandler(key)
			return uc.handleErr(err, key)
		}

		// Finally, if no error occurs we forget this item so it does not
		// get queued again until another change happens.
		uc.workqueue.Forget(obj)
		log.Debugf("%s: Successfully synced '%s'", upgradeControllerName, key)
		return nil
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
// updated and sets that as the current node on the cluster resource.
func (uc *UpgradeController) upgradeSyncHandler(key string) error {
	_, _, name, _ := tools.SplitMetaResourceNamespaceKeyFunc(key)
	upgrade, err := uc.upgradeLister.ClusterUpgrades(constants.ContainershipNamespace).Get(name)
	// If upgrade has already been fully processed and either Successed or Failed
	// we don't need to do anything
	// This should never happen since we're only listening for Add events but
	// let's be safe
	if isUpgradeDone(upgrade) {
		return nil
	}

	// if cluster is not in upgraded state
	node := uc.getNextNode(upgrade)
	if node == nil {
		return nil
	}

	currentUpgrade := upgrade.DeepCopy()
	currentUpgrade.Spec.CurrentNode = node.Name
	currentUpgrade.Spec.Status = provisioncsv3.UpgradeInProgress
	_, err = uc.csclientset.ContainershipProvisionV3().ClusterUpgrades(constants.ContainershipNamespace).Update(currentUpgrade)

	uc.currentUpgradeName = currentUpgrade.Name

	return err
}

// nodeIsDone returns true if the given node is finished upgrading, else false.
// Note that "done" may be success or failure.
func nodeIsDone(node *corev1.Node) (bool, error) {
	upgradeAnnotation, err := tools.GetNodeUpgradeAnnotation(node)
	switch {
	case err != nil:
		return false, err
	case upgradeAnnotation == nil:
		return false, nil
	case upgradeAnnotation.Status == provisioncsv3.UpgradeInProgress:
		return false, nil
	case upgradeAnnotation.Status == provisioncsv3.UpgradeSuccess:
		return true, nil
	case upgradeAnnotation.Status == provisioncsv3.UpgradeFailed:
		return true, nil
	}

	return false, nil
}

func (uc *UpgradeController) nodeSyncHandler(key string) error {
	_, _, name, _ := tools.SplitMetaResourceNamespaceKeyFunc(key)

	node, err := uc.nodeLister.Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			// Node is no longer around so no need to upgrade
			return nil
		}

		return err
	}

	currentUpgrade, err := uc.upgradeLister.ClusterUpgrades(constants.ContainershipNamespace).
		Get(uc.currentUpgradeName)
	if err != nil {
		if errors.IsNotFound(err) {
			// Upgrade is no longer around so no need to upgrade
			return nil
		}

		return err
	}

	if !isCurrentNode(currentUpgrade, node) {
		// Nothing to do
		return nil
	}

	done, err := nodeIsDone(node)
	if err != nil {
		return err
	}

	if done {
		log.Infof("%s: Node %s finished upgrading", upgradeControllerName, node.Name)

		next := uc.getNextNode(currentUpgrade)

		currentUpgradeCopy := currentUpgrade.DeepCopy()
		if next == nil {
			// No more nodes to upgrade, so finish up
			currentUpgradeCopy.Spec.Status = uc.getFinalUpgradeStatus(currentUpgradeCopy)
			currentUpgradeCopy.Spec.CurrentNode = ""
		} else {
			log.Infof("%s: Marking node %s for upgrade", upgradeControllerName, next.Name)
			currentUpgradeCopy.Spec.CurrentNode = next.Name
		}

		_, err = uc.csclientset.ContainershipProvisionV3().ClusterUpgrades(constants.ContainershipNamespace).Update(currentUpgradeCopy)
		if err != nil {
			return err
		}
	}

	return nil
}

func (uc *UpgradeController) getFinalUpgradeStatus(cup *provisioncsv3.ClusterUpgrade) provisioncsv3.UpgradeStatus {
	nodes, _ := uc.nodeLister.List(getAllNodesSelector(cup.Spec.LabelSelector))
	for _, node := range nodes {
		upgradeAnnotation, _ := tools.GetNodeUpgradeAnnotation(node)
		if upgradeAnnotation.Status == provisioncsv3.UpgradeFailed {
			return provisioncsv3.UpgradeFailed
		}
	}

	return provisioncsv3.UpgradeSuccess
}

// enqueueUpgrade enqueues an upgrade
func (uc *UpgradeController) enqueueUpgrade(obj interface{}) {
	key, err := tools.MetaResourceNamespaceKeyFunc("upgrade", obj)
	if err != nil {
		log.Error(err)
		return
	}

	uc.workqueue.AddRateLimited(key)
}

// enqueueNode enqueues a node
func (uc *UpgradeController) enqueueNode(obj interface{}) {
	key, err := tools.MetaResourceNamespaceKeyFunc("node", obj)
	if err != nil {
		log.Error(err)
		return
	}

	uc.workqueue.AddRateLimited(key)
}

// isUpgradeDone returns true if an upgrade has already been fully processed and has
// the status of either Successed or Failed
func isUpgradeDone(cup *provisioncsv3.ClusterUpgrade) bool {
	if cup.Spec.Status == provisioncsv3.UpgradeSuccess || cup.Spec.Status == provisioncsv3.UpgradeFailed {
		return true
	}

	return false
}

// isCurrentNode checks to see if the node being looked at is the current node being processed
func isCurrentNode(cup *provisioncsv3.ClusterUpgrade, node *corev1.Node) bool {
	return cup.Spec.CurrentNode == node.Name
}

// getNextNode finds the next node to start upgrading or nil if all nodes are
// finished upgrading.
func (uc *UpgradeController) getNextNode(cup *provisioncsv3.ClusterUpgrade) *corev1.Node {
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

// isNext returns true if the given node can go next for this upgrade, else false.
func isNext(cup *provisioncsv3.ClusterUpgrade, node *corev1.Node) bool {
	return !tools.NodeIsTargetKubernetesVersion(cup, node) && !isCurrentNode(cup, node)
}

// addCustomLabelSelectors appends additional selectors to the given selector.
func addCustomLabelSelectors(selector labels.Selector, lss []provisioncsv3.LabelSelectorSpec) labels.Selector {
	for _, ls := range lss {
		nr, _ := labels.NewRequirement(ls.Label, ls.Operator, ls.Value)
		selector = selector.Add(*nr)
	}

	return selector
}

// getAllNodesSelector gets a selector for all Containership-managed nodes plus
// any additional selectors specified as an argument.
func getAllNodesSelector(lss []provisioncsv3.LabelSelectorSpec) labels.Selector {
	selector := constants.GetContainershipManagedSelector()
	selector = addCustomLabelSelectors(selector, lss)
	return selector
}

// getMasterSelector gets a selector for all Containership-managed master
// nodes plus any additional selectors specified as an argument.
func getMasterSelector(lss []provisioncsv3.LabelSelectorSpec) labels.Selector {
	masterLabelExists, _ := labels.NewRequirement("node-role.kubernetes.io/master", selection.Exists, []string{})
	selector := constants.GetContainershipManagedSelector()
	selector = selector.Add(*masterLabelExists)
	selector = addCustomLabelSelectors(selector, lss)
	return selector
}

// getWorkerSelector gets a selector for all Containership-managed worker
// nodes plus any additional selectors specified as an argument.
func getWorkerSelector(lss []provisioncsv3.LabelSelectorSpec) labels.Selector {
	masterLabelDNE, _ := labels.NewRequirement("node-role.kubernetes.io/master", selection.DoesNotExist, []string{})
	selector := constants.GetContainershipManagedSelector()
	selector = selector.Add(*masterLabelDNE)
	selector = addCustomLabelSelectors(selector, lss)
	return selector
}
