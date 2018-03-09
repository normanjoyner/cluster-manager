package agent

import (
	"fmt"
	"time"

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

	containershipv3 "github.com/containership/cloud-agent/pkg/apis/containership.io/v3"
	csclientset "github.com/containership/cloud-agent/pkg/client/clientset/versioned"
	csinformers "github.com/containership/cloud-agent/pkg/client/informers/externalversions"
	cslisters "github.com/containership/cloud-agent/pkg/client/listers/containership.io/v3"
)

const (
	maxRetriesUpgradeController = 5
)

// UpgradeController is the agent controller which watches for ClusterUpgrade updates
// and writes update script to host when it is that specific agents turn to update
type UpgradeController struct {
	clientset     csclientset.Interface
	kubeclientset kubernetes.Interface

	upgradeLister  cslisters.ClusterUpgradeLister
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
	upgradeInformer := csInformerFactory.Containership().V3().ClusterUpgrades()
	nodeInformer := kubeInformerFactory.Core().V1().Nodes()

	// All event handlers simply add to a workqueue to be processed by a worker
	upgradeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(old, new interface{}) {
			newUpgrade := new.(*containershipv3.ClusterUpgrade)
			oldUpgrade := old.(*containershipv3.ClusterUpgrade)
			if oldUpgrade.ResourceVersion == newUpgrade.ResourceVersion || newUpgrade.Spec.CurrentNode != env.NodeName() {
				// This must be a syncInterval update, or does not apply to the
				// host that the agent is running on
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

		// Run the syncHandler, passing it the namespace/name string of the
		// User resource to be synced.
		err := uc.syncHandler(key)
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
		return fmt.Errorf("error syncing '%v': %s. has been resynced %v times", key, err.Error(), uc.workqueue.NumRequeues(key))
	}

	uc.workqueue.Forget(key)
	log.Infof("Dropping %v out of the queue: %v", key, err)
	return err
}

// enqueueUpgrade enqueues the key for a given upgrade - it should not be called
// for other object types
func (uc *UpgradeController) enqueueUpgrade(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		log.Error(err)
		return
	}
	uc.workqueue.AddRateLimited(key)
}

// syncHandler looks at the current state of the system and decides how to act.
// For upgrade that means writing the upgrade script to the directory that is being
// watched by the systemd upgrade process.
func (uc *UpgradeController) syncHandler(key string) error {
	log.Infof("Upgrade processing: key=%s", key)
	_, name, _ := cache.SplitMetaNamespaceKey(key)
	// We only want to act on cluster upgrades that are from the containership-core
	// namespace to make sure they are valid and were created by containership cloud
	upgrade, err := uc.upgradeLister.ClusterUpgrades(constants.ContainershipNamespace).Get(name)
	if err != nil {
		return err
	}

	node, _ := uc.nodeLister.Get(env.NodeName())
	if tools.NodeIsTargetKubernetesVersion(upgrade, node) == true {
		return nil
	}

	//// TODO figure out how we are getting script from cloud
	// currently we are talking about a path that looks like
	// /organization/:organization_id/cluster/:cluster_id/host/:host_id
	// Step 1: Fetch the upgrade script from Cloud
	script := []byte("#!/bin/bash\necho 'hallo'\n")

	targetVersion := upgrade.Spec.TargetKubernetesVersion
	upgradeID := upgrade.Spec.ID
	// Step 2: Execute the upgrade script
	return upgradescript.Write(script, targetVersion, upgradeID)
}
