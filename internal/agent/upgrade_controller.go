package agent

import (
	"encoding/json"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"

	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	corelistersv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/containership/cloud-agent/internal/constants"
	"github.com/containership/cloud-agent/internal/env"
	"github.com/containership/cloud-agent/internal/log"
	"github.com/containership/cloud-agent/internal/resources/upgradescript"
	"github.com/containership/cloud-agent/internal/tools"

	provisioncsv3 "github.com/containership/cloud-agent/pkg/apis/provision.containership.io/v3"
	csclientset "github.com/containership/cloud-agent/pkg/client/clientset/versioned"
	csinformers "github.com/containership/cloud-agent/pkg/client/informers/externalversions"
	pcslisters "github.com/containership/cloud-agent/pkg/client/listers/provision.containership.io/v3"
)

const (
	upgradeControllerName = "upgradeAgent"

	maxRetriesUpgradeController = 5
)

// UpgradeController is the agent controller which watches for ClusterUpgrade updates
// and writes update script to host when it is that specific agents turn to update
type UpgradeController struct {
	clientset     csclientset.Interface
	kubeclientset kubernetes.Interface

	upgradeLister  pcslisters.ClusterUpgradeLister
	upgradesSynced cache.InformerSynced
	nodeLister     corelistersv1.NodeLister
	nodesSynced    cache.InformerSynced

	workqueue workqueue.RateLimitingInterface
}

// NewUpgradeController creates a new agent UpgradeController
func NewUpgradeController(
	clientset csclientset.Interface,
	csInformerFactory csinformers.SharedInformerFactory,
	kubeclientset kubernetes.Interface,
	kubeInformerFactory kubeinformers.SharedInformerFactory) *UpgradeController {

	uc := &UpgradeController{
		clientset:     clientset,
		kubeclientset: kubeclientset,
		workqueue:     workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "UpgradeAgent"),
	}

	// Create an informer from the factory so that we share the underlying
	// cache with other controllers
	upgradeInformer := csInformerFactory.ContainershipProvision().V3().ClusterUpgrades()
	nodeInformer := kubeInformerFactory.Core().V1().Nodes()

	// All event handlers simply add to a workqueue to be processed by a worker
	upgradeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(old, new interface{}) {
			newUpgrade := new.(*provisioncsv3.ClusterUpgrade)
			if newUpgrade.Spec.CurrentNode != env.NodeName() {
				// Not the current node so nothing to do
				return
			}
			uc.enqueueUpgrade(new)
		},
	})

	uc.upgradeLister = upgradeInformer.Lister()
	uc.upgradesSynced = upgradeInformer.Informer().HasSynced
	uc.nodeLister = nodeInformer.Lister()
	uc.nodesSynced = nodeInformer.Informer().HasSynced

	return uc
}

// Run kicks off the Controller with the given number of workers to process the
// workqueue
func (uc *UpgradeController) Run(numWorkers int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer uc.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	log.Info("Starting Upgrade controller")

	log.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, uc.upgradesSynced, uc.nodesSynced); !ok {
		return fmt.Errorf("Failed to wait for caches to sync")
	}

	log.Info("Starting upgrade workers")
	// Launch numWorkers workers to process Upgrade resource
	for i := 0; i < numWorkers; i++ {
		go wait.Until(uc.runWorker, time.Second, stopCh)
	}

	log.Info("Started upgrade workers")
	<-stopCh
	log.Info("Shutting down upgrade controller")

	return nil
}

// runWorker continually requests that the next queue item be processed
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

		err := uc.upgradeSyncHandler(key)
		return uc.handleErr(err, key)
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
// the resource is moved off the queue
func (uc *UpgradeController) handleErr(err error, key interface{}) error {
	if err == nil {
		uc.workqueue.Forget(key)
		return nil
	}

	if uc.workqueue.NumRequeues(key) < maxRetriesUpgradeController {
		uc.workqueue.AddRateLimited(key)
		return fmt.Errorf("error syncing '%v': %s. Has been resynced %v times", key, err.Error(), uc.workqueue.NumRequeues(key))
	}

	uc.workqueue.Forget(key)
	log.Infof("Dropping %v out of the queue: %v", key, err)
	return err
}

// enqueueUpgrade enqueues an upgrade
func (uc *UpgradeController) enqueueUpgrade(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Error(err)
		return
	}

	uc.workqueue.AddRateLimited(key)
}

// upgradeSyncHandler looks at the current state of the system and decides how to act.
// For upgrade that means writing the upgrade script to the directory that is being
// watched by the systemd upgrade process.
func (uc *UpgradeController) upgradeSyncHandler(key string) error {
	log.Infof("Upgrade processing: key=%q", key)
	_, name, _ := cache.SplitMetaNamespaceKey(key)
	// We only want to act on cluster upgrades that are from the containership-core
	// namespace to make sure they are valid and were created by containership cloud
	upgrade, err := uc.upgradeLister.ClusterUpgrades(constants.ContainershipNamespace).Get(name)
	if err != nil {
		return err
	}

	node, _ := uc.nodeLister.Get(env.NodeName())

	upgradeAnnotation, err := tools.GetNodeUpgradeAnnotation(node)
	if err != nil {
		return err
	}

	switch {
	case upgradeAnnotation == nil:
		// Upgrade hasn't started or we're in an unknown state
		if tools.NodeIsTargetKubernetesVersion(upgrade, node) {
			// Nothing to do since we're at the desired version
			return nil
		}

		// Kick off upgrade and mark status as InProgress
		return uc.startUpgrade(node, upgrade)

	case upgradeAnnotation.Status == provisioncsv3.UpgradeInProgress:
		if tools.NodeIsTargetKubernetesVersion(upgrade, node) {
			// We must be done upgrading - presumably it succeeded because
			// we're at the version we expect and we didn't time out.
			// Finish the upgrade with a Success status.
			return uc.finishUpgradeWithStatus(node, provisioncsv3.UpgradeSuccess)
		}

	case upgradeAnnotation.Status == provisioncsv3.UpgradeSuccess:
	case upgradeAnnotation.Status == provisioncsv3.UpgradeFailed:
	default:
		// Nothing to do
		return nil
	}

	return nil
}

func (uc *UpgradeController) startUpgrade(node *corev1.Node, upgrade *provisioncsv3.ClusterUpgrade) error {
	log.Info("Beginning upgrade process")

	// Step 1: Fetch the upgrade script from Cloud
	log.Info("Downloading upgrade script")
	script, err := uc.downloadUpgradeScript()
	if err != nil {
		log.Error("Download upgrade script failed:", err)
		return err
	}

	// Step 2: Mark node as in progress
	log.Info("Setting node upgrade status to InProgress")
	err = uc.writeNodeUpgradeAnnotation(node,
		provisioncsv3.NodeUpgradeAnnotation{
			upgrade.Spec.TargetKubernetesVersion,
			provisioncsv3.UpgradeInProgress,
			time.Now().UTC(),
		})
	if err != nil {
		log.Error("Node annotation write failed:", err)
		return err
	}

	// Step 3: Execute the upgrade script
	log.Info("Writing upgrade script")
	targetVersion := upgrade.Spec.TargetKubernetesVersion
	upgradeID := upgrade.Spec.ID
	return upgradescript.Write(script, targetVersion, upgradeID)
}

// finishUpgradeWithStatus performs any necessary cleanup and posts the updated
// status for this node.
func (uc *UpgradeController) finishUpgradeWithStatus(node *corev1.Node,
	status provisioncsv3.UpgradeStatus) error {
	log.Info("Setting node upgrade status to %s", string(status))

	// Remove the `current` file first so regardless of any failures after this point
	// we'll be able to retry if needed by writing a new `current`
	if err := upgradescript.RemoveCurrent(); err != nil {
		// There's no good option for handling this, so just continue instead of
		// failing the upgrade.
		log.Error("Could not remove `current` upgrade file:", err)
	}

	upgradeAnnotation, err := tools.GetNodeUpgradeAnnotation(node)
	if err != nil {
		return err
	}

	return uc.writeNodeUpgradeAnnotation(node,
		provisioncsv3.NodeUpgradeAnnotation{
			upgradeAnnotation.ClusterVersion,
			status,
			upgradeAnnotation.StartTime,
		})
}

func (uc *UpgradeController) downloadUpgradeScript() ([]byte, error) {
	//// TODO figure out how we are getting script from cloud
	// currently we are talking about a path that looks like
	// /organization/:organization_id/cluster/:cluster_id/host/:host_id
	script := []byte("#!/bin/bash\necho 'hallo'\n")
	return script, nil
}

// writeNodeUpgradeAnnotation writes the node upgrade annotation
// struct to the node
func (uc *UpgradeController) writeNodeUpgradeAnnotation(node *corev1.Node, upgradeAnnotation provisioncsv3.NodeUpgradeAnnotation) error {
	node = node.DeepCopy()

	annotBytes, err := json.Marshal(upgradeAnnotation)
	if err != nil {
		return err
	}

	annotations := node.ObjectMeta.Annotations
	annotations[constants.NodeUpgradeAnnotationKey] = string(annotBytes)

	_, err = uc.kubeclientset.Core().Nodes().Update(node)

	return err
}
