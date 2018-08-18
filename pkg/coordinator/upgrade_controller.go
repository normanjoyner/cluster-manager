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

	"github.com/containership/cloud-agent/pkg/constants"
	"github.com/containership/cloud-agent/pkg/log"
	"github.com/containership/cloud-agent/pkg/tools"

	provisioncsv3 "github.com/containership/cloud-agent/pkg/apis/provision.containership.io/v3"
	csclientset "github.com/containership/cloud-agent/pkg/client/clientset/versioned"
	csinformers "github.com/containership/cloud-agent/pkg/client/informers/externalversions"
	pcslisters "github.com/containership/cloud-agent/pkg/client/listers/provision.containership.io/v3"
)

const (
	upgradeControllerName = "UpgradeController"

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
		uc.upgradesSynced,
		uc.nodesSynced); !ok {
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

// enqueueNodeAfterDelay enqueues a node after a given delay. It is intended to
// be called asynchronously.
func (uc *UpgradeController) enqueueNodeAfterDelay(node *corev1.Node, d time.Duration) {
	timer := time.NewTimer(d)
	<-timer.C
	log.Debugf("Enqueuing node %q after %v", node.Name, d)
	uc.enqueueNode(node)
}

// upgradeSyncHandler looks at the UpgradeCluster resource and if it is in a
// completed state return. Otherwise, it finds the next node that should be
// upgraded and sets that as the current node on the cluster resource.
func (uc *UpgradeController) upgradeSyncHandler(key string) error {
	_, _, name, _ := tools.SplitMetaResourceNamespaceKeyFunc(key)
	upgrade, err := uc.upgradeLister.ClusterUpgrades(constants.ContainershipNamespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			// Upgrade is no longer around so nothing to do
			return nil
		}

		return err
	}

	switch upgrade.Spec.Type {
	case provisioncsv3.UpgradeTypeKubernetes:
		// TODO in the future we should cleanly separate logic for different
		// types as needed. For now, we can just assume Kubernetes upgrades
		// from this point forward.
		uc.recorder.Eventf(upgrade, corev1.EventTypeNormal, "Accepted", "Upgrade type %q accepted for processing", upgrade.Spec.Type)
		break
	case provisioncsv3.UpgradeTypeEtcd:
		fallthrough
	default:
		// Record an error but return nil so we don't retry since there's nothing we can do
		uc.recorder.Eventf(upgrade, corev1.EventTypeWarning, "Ignore", "Unsupported upgrade type %q", upgrade.Spec.Type)
		return nil
	}

	// If upgrade has already been fully processed and either Successed or Failed
	// we don't need to do anything. We're only listening to Add events so this
	// should be unlikely, but it can happen e.g. if coordinator restarts.
	if isUpgradeDone(upgrade) {
		return nil
	}

	existingUpgrade, _ := uc.getCurrentUpgrade()
	if existingUpgrade != nil {
		// There's already an upgrade in-progress. This should
		// never happen, but ignore it to be safe.
		return nil
	}

	// Cluster is not in an upgraded state, so kick off the upgrade process
	// by marking the first applicable node as in-progress.
	node := uc.getNextNode(upgrade)
	if node == nil {
		// We're already at the target version, so just finish the upgrade.
		return uc.finishUpgrade(upgrade)
	}

	return uc.startUpgradeForNode(upgrade, node)
}

// nodeSyncHandler surveys the system state and determines which node, if any,
// is next to upgrade.
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

	currentUpgrade, err := uc.getCurrentUpgrade()
	if err != nil {
		return err
	}
	if currentUpgrade == nil {
		// No active upgrades, so nothing to do
		return nil
	}

	if !isCurrentNode(currentUpgrade, node) {
		// Current upgrade does not apply to this node, so nothing to do
		return nil
	}

	// Starting here, we may mutate the upgrade object and make decisions based
	// on that mutated state. To avoid race conditions, we must only operate on
	// a copy of the state.
	currentUpgrade = currentUpgrade.DeepCopy()

	nodeIsTargetVersion := tools.NodeIsTargetKubernetesVersion(currentUpgrade, node)
	nodeTimedOut := false
	if !nodeIsTargetVersion {
		// Check for timeout
		startTime, _ := time.Parse(time.UnixDate, currentUpgrade.Spec.Status.CurrentStartTime)
		elapsed := time.Since(startTime)
		if elapsed.Seconds() >= float64(currentUpgrade.Spec.NodeTimeoutSeconds) {
			nodeTimedOut = true
		}
	}

	readyToMoveOn := nodeIsTargetVersion && tools.NodeIsReady(node)
	if !nodeTimedOut && !readyToMoveOn {
		// Upgrade is still processing, nothing to no
		return nil
	}

	// Current node is done upgrading, decide how to proceed
	// This map shouldn't be nil since the map should have been created when
	// the upgrade was kicked off, but let's be safe.
	if currentUpgrade.Spec.Status.NodeStatuses == nil {
		currentUpgrade.Spec.Status.NodeStatuses = make(map[string]provisioncsv3.UpgradeStatus)
	}

	// Mark this node as done with appropriate status
	if nodeTimedOut {
		uc.recorder.Eventf(currentUpgrade, corev1.EventTypeWarning, "NodeUpgradeFailure", "Node %q upgrade timed out", node.Name)
		currentUpgrade.Spec.Status.NodeStatuses[node.Name] = provisioncsv3.UpgradeFailed
	} else {
		uc.recorder.Eventf(currentUpgrade, corev1.EventTypeNormal, "NodeUpgradeSuccess", "Node %q upgrade succeeded", node.Name)
		currentUpgrade.Spec.Status.NodeStatuses[node.Name] = provisioncsv3.UpgradeSuccess
	}

	// Post the node status as running regardless of success or failure because
	// this cloud status is only used to determine if a node is running or not.
	nodeID := node.Labels[constants.ContainershipNodeIDLabelKey]
	tryPostNodeCloudStatusRunning(nodeID)

	next := uc.getNextNode(currentUpgrade)
	if next == nil {
		// No more nodes to upgrade, so finish up
		err = uc.finishUpgrade(currentUpgrade)
	} else {
		// Kick off the next node upgrade
		err = uc.startUpgradeForNode(currentUpgrade, next)
	}

	return err
}

// tryPostNodeCloudStatusRunning tries to post given node status to cloud as
// running and silently ignores any errors that may occur.
func tryPostNodeCloudStatusRunning(nodeID string) {
	status := NodeCloudStatusMessage{
		Status: NodeCloudStatus{
			Type:    NodeCloudStatusRunning,
			Percent: "1",
		},
	}
	_ = PostNodeCloudStatusMessageWithRetry(nodeID, &status, 3)
}

// updateClusterUpgradeStatus posts an updated status for the given upgrade object
func (uc *UpgradeController) updateClusterUpgradeStatus(cup *provisioncsv3.ClusterUpgrade,
	status *provisioncsv3.ClusterUpgradeStatusSpec) error {
	cup = cup.DeepCopy()
	status.DeepCopyInto(&cup.Spec.Status)
	_, err := uc.csclientset.ContainershipProvisionV3().ClusterUpgrades(constants.ContainershipNamespace).Update(cup)
	return err
}

// finishUpgrade finishes the given upgrade by posting back the final upgrade status.
func (uc *UpgradeController) finishUpgrade(cup *provisioncsv3.ClusterUpgrade) error {
	clusterStatus := getFinalUpgradeStatus(cup)

	uc.recorder.Eventf(cup, corev1.EventTypeNormal, "ClusterUpgradeComplete", "Cluster upgrade completed with status %q", clusterStatus)

	return uc.updateClusterUpgradeStatus(cup, &provisioncsv3.ClusterUpgradeStatusSpec{
		ClusterStatus:    clusterStatus,
		NodeStatuses:     cup.Spec.Status.NodeStatuses,
		CurrentNode:      "",
		CurrentStartTime: "",
	})
}

// startUpgradeForNode kicks off the upgrade process for the given node by updating
// the ClusterUpgrade CRD appropriately.
func (uc *UpgradeController) startUpgradeForNode(cup *provisioncsv3.ClusterUpgrade, node *corev1.Node) error {
	uc.recorder.Eventf(cup, corev1.EventTypeNormal, "NodeInProgress", "Marking node %q for upgrade", node.Name)

	nodeStatuses := cup.Spec.Status.NodeStatuses
	if nodeStatuses == nil {
		nodeStatuses = make(map[string]provisioncsv3.UpgradeStatus)
	}
	nodeStatuses[node.Name] = provisioncsv3.UpgradeInProgress

	err := uc.updateClusterUpgradeStatus(cup, &provisioncsv3.ClusterUpgradeStatusSpec{
		ClusterStatus:    provisioncsv3.UpgradeInProgress,
		NodeStatuses:     nodeStatuses,
		CurrentNode:      node.Name,
		CurrentStartTime: time.Now().UTC().Format(time.UnixDate),
	})

	// Ensure the syncHandler is called for this node in the future in order
	// to check for timeout
	delay := time.Second * time.Duration(cup.Spec.NodeTimeoutSeconds)
	go uc.enqueueNodeAfterDelay(node, delay)

	return err
}

// getCurrentUpgrade returns the current in-progress upgrade or nil if no upgrade
// is in-progress.
func (uc *UpgradeController) getCurrentUpgrade() (*provisioncsv3.ClusterUpgrade, error) {
	upgrades, err := uc.upgradeLister.ClusterUpgrades(constants.ContainershipNamespace).
		List(constants.GetContainershipManagedSelector())
	if err != nil {
		return nil, err
	}

	for _, upgrade := range upgrades {
		if upgrade.Spec.Status.ClusterStatus == provisioncsv3.UpgradeInProgress {
			return upgrade, nil
		}
	}

	return nil, nil
}

// getFinalUpgradeStatus returns the final upgrade status for the given cluster
// upgrade based on the individual node statuses. If no node statuses are
// present (e.g. because this is called for a ClusterUpgrade for which we're
// already at the target version), then this function returns Success
func getFinalUpgradeStatus(cup *provisioncsv3.ClusterUpgrade) provisioncsv3.UpgradeStatus {
	for _, status := range cup.Spec.Status.NodeStatuses {
		if status == provisioncsv3.UpgradeFailed {
			return provisioncsv3.UpgradeFailed
		}
	}

	return provisioncsv3.UpgradeSuccess
}

// isUpgradeDone returns true if an upgrade has already been fully processed and has
// the status of either Successed or Failed
func isUpgradeDone(cup *provisioncsv3.ClusterUpgrade) bool {
	if cup.Spec.Status.ClusterStatus == provisioncsv3.UpgradeSuccess ||
		cup.Spec.Status.ClusterStatus == provisioncsv3.UpgradeFailed {
		return true
	}

	return false
}

// isCurrentNode checks to see if the node being looked at is the current node being processed
func isCurrentNode(cup *provisioncsv3.ClusterUpgrade, node *corev1.Node) bool {
	return cup.Spec.Status.CurrentNode == node.Name
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
	return !isCurrentNode(cup, node) &&
		!tools.NodeIsTargetKubernetesVersion(cup, node) &&
		!nodeHasFinishedStatus(cup, node)
}

// nodeHasFinishedStatus returns true if the given node has a "finished" status,
// i.e. its status exists and it is  either a success or failed status.
func nodeHasFinishedStatus(cup *provisioncsv3.ClusterUpgrade, node *corev1.Node) bool {
	return cup.Spec.Status.NodeStatuses[node.Name] == provisioncsv3.UpgradeSuccess ||
		cup.Spec.Status.NodeStatuses[node.Name] == provisioncsv3.UpgradeFailed
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
