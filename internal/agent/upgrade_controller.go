package agent

import (
	"fmt"
	"io/ioutil"
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
	"github.com/containership/cloud-agent/internal/request"
	"github.com/containership/cloud-agent/internal/resources/upgradescript"

	provisioncsv3 "github.com/containership/cloud-agent/pkg/apis/provision.containership.io/v3"
	csclientset "github.com/containership/cloud-agent/pkg/client/clientset/versioned"
	csinformers "github.com/containership/cloud-agent/pkg/client/informers/externalversions"
	pcslisters "github.com/containership/cloud-agent/pkg/client/listers/provision.containership.io/v3"
)

const (
	upgradeControllerName = "UpgradeAgentController"

	maxRetriesUpgradeController = 5
)

const (
	// TODO finalize this - current version is just for rough testing
	nodeUpgradeScriptEndpointTemplate = "/organizations/{{.OrganizationID}}/clusters/{{.ClusterID}}/nodes/{{.NodeName}}-upgrade.sh"
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
		workqueue:     workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), upgradeControllerName),
	}

	// Create an informer from the factory so that we share the underlying
	// cache with other controllers
	upgradeInformer := csInformerFactory.ContainershipProvision().V3().ClusterUpgrades()
	nodeInformer := kubeInformerFactory.Core().V1().Nodes()

	// All event handlers simply add to a workqueue to be processed by a worker
	upgradeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(old, new interface{}) {
			oldUpgrade := old.(*provisioncsv3.ClusterUpgrade)
			newUpgrade := new.(*provisioncsv3.ClusterUpgrade)
			if oldUpgrade.ResourceVersion == newUpgrade.ResourceVersion ||
				newUpgrade.Spec.Status.CurrentNode != env.NodeName() {
				// Just a syncInterval update or this update does not apply to us
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

// enqueueNode enqueues a node
func (uc *UpgradeController) enqueueNode(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Error(err)
		return
	}

	uc.workqueue.AddRateLimited(key)
}

// syncHandler looks at the current state of the system and decides how to act.
// For upgrade that means writing the upgrade script to the directory that is being
// watched by the systemd upgrade process.
func (uc *UpgradeController) syncHandler(key string) error {
	log.Debugf("%s: processing key=%q", upgradeControllerName, key)

	_, name, _ := cache.SplitMetaNamespaceKey(key)

	upgrade, err := uc.upgradeLister.ClusterUpgrades(constants.ContainershipNamespace).Get(name)
	if err != nil {
		return err
	}
	if upgrade == nil {
		// Upgrade no longer exists, nothing to do
		return nil
	}

	switch upgrade.Spec.Type {
	case provisioncsv3.UpgradeTypeKubernetes:
		// TODO in the future we should cleanly separate logic for different
		// types as needed. For now, we can just assume Kubernetes upgrades
		// from this point forward.
		break
	case provisioncsv3.UpgradeTypeEtcd:
		fallthrough
	default:
		// Log an error but return nil so we don't retry since there's nothing we can do
		log.Errorf("%s: ignoring unsupported upgrade type %q", upgradeControllerName, upgrade.Spec.Type)
		return nil
	}

	if upgrade.Spec.Status.NodeStatuses[env.NodeName()] != provisioncsv3.UpgradeInProgress {
		// It's not our turn to do anything
		return nil
	}

	upgradeType := upgrade.Spec.Type
	targetVersion := upgrade.Spec.TargetVersion
	upgradeID := upgrade.Spec.ID

	if upgradescript.Exists(upgradeType, targetVersion, upgradeID) {
		return nil
	}

	return uc.startUpgrade(upgrade)
}

// startUpgrade kicks off the upgrade process by downloading and writing the
// upgrade script as well as updating the current node's upgrade status.
func (uc *UpgradeController) startUpgrade(upgrade *provisioncsv3.ClusterUpgrade) error {
	log.Info("Beginning upgrade process")

	// Step 1: Fetch the upgrade script from Cloud
	log.Info("Downloading upgrade script")
	script, err := uc.downloadUpgradeScript()
	if err != nil {
		log.Error("Download upgrade script failed:", err)
		return err
	}

	// Step 2: Execute the upgrade script
	log.Info("Writing upgrade script")
	upgradeType := upgrade.Spec.Type
	targetVersion := upgrade.Spec.TargetVersion
	upgradeID := upgrade.Spec.ID
	return upgradescript.Write(script, upgradeType, targetVersion, upgradeID)
}

// finishUpgrade performs any necessary cleanup and finalizes the upgrade for this node
func (uc *UpgradeController) finishUpgrade(node *corev1.Node) {
	// Remove the `current` file first so regardless of any failures after this point
	// we'll be able to retry if needed by writing a new `current`
	if err := upgradescript.RemoveCurrent(); err != nil {
		// There's no good option for handling this, so just continue instead of
		// failing the upgrade.
		log.Error("Could not remove `current` upgrade file:", err)
	}
}

// downloadUpgradeScript downloads the upgrade script for this node
func (uc *UpgradeController) downloadUpgradeScript() ([]byte, error) {
	req, err := request.New(request.CloudServiceProvision,
		nodeUpgradeScriptEndpointTemplate,
		"GET",
		nil)
	if err != nil {
		return nil, err
	}

	resp, err := req.MakeRequest()
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}
