package coordinator

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
)

const (
	// Type of agent that runs this controller
	controllerName = "ContainershipController"
	// number of times an object will be requeued if there is an error
	maxRetriesContrainershipController = 5
)

// ContainershipController is the controller implementation for the containership
// setup
type ContainershipController struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface

	namespacesLister corelistersv1.NamespaceLister
	namespacesSynced cache.InformerSynced

	serviceAccountsLister corelistersv1.ServiceAccountLister
	serviceAccountsSynced cache.InformerSynced

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

// NewContainershipController returns a new containership controller
func NewContainershipController(kubeclientset kubernetes.Interface, kubeInformerFactory kubeinformers.SharedInformerFactory) *ContainershipController {
	c := &ContainershipController{
		kubeclientset: kubeclientset,
		workqueue:     workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Containership"),
		recorder:      tools.CreateAndStartRecorder(kubeclientset, controllerName),
	}
	// Instantiate resource informers
	namespaceInformer := kubeInformerFactory.Core().V1().Namespaces()
	serviceAccountInformer := kubeInformerFactory.Core().V1().ServiceAccounts()

	log.Info(controllerName, ": Setting up event handlers")

	// namespace informer listens for add events an queues the namespace
	namespaceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueueNamespace,
	})
	// service account informer listens for the delete event and queues the service account
	serviceAccountInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: c.deleteServiceAccount,
	})

	// Listers are used for cache inspection and Synced functions
	// are used to wait for cache synchronization
	c.namespacesLister = namespaceInformer.Lister()
	c.namespacesSynced = namespaceInformer.Informer().HasSynced
	c.serviceAccountsLister = serviceAccountInformer.Lister()
	c.serviceAccountsSynced = serviceAccountInformer.Informer().HasSynced
	return c
}

func (c *ContainershipController) deleteServiceAccount(obj interface{}) {
	if constants.IsContainershipManaged(obj) {
		c.enqueueServiceAccount(obj)
	}
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *ContainershipController) Run(numWorkers int, stopCh chan struct{}) {
	defer runtime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	log.Info(controllerName, ": Starting controller")

	if ok := cache.WaitForCacheSync(
		stopCh,
		c.serviceAccountsSynced,
		c.namespacesSynced); !ok {
		// If this channel is unable to wait for caches to sync we stop both
		// the containership controller, and the registry controller
		close(stopCh)
		log.Error("failed to wait for caches to sync")
	}

	log.Info(controllerName, ": Starting workers")
	// Launch numWorkers amount of workers to process resources
	for i := 0; i < numWorkers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	log.Info(controllerName, ": Started workers")
	<-stopCh
	log.Info(controllerName, ": Shutting down workers")
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *ContainershipController) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the appropriate syncHandler.
func (c *ContainershipController) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form kind/namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workqueue.Forget(obj)
			log.Errorf("expected string in workqueue but got %#v", obj)
			return nil
		}

		kind, namespace, name, err := tools.SplitMetaResourceNamespaceKeyFunc(key)
		if err != nil {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workqueue.Forget(obj)
			log.Errorf("key is in incorrect format to process %#v", obj)
			return nil
		}

		// If namespace is empty but it has a name it is a namespace resource,
		// and we want to be able to check it for being terminating or terminated
		// like any other resource
		if namespace == "" && name != "" {
			namespace = name
		}

		terminatingOrTerminated, err := checkNamespace(c.namespacesLister, namespace)
		if err != nil {
			return err
		}

		if terminatingOrTerminated == true {
			log.Infof("%s: Namespace '%s' for %s in work queue does not exist", controllerName, namespace, kind)
			return nil
		}

		// Run the needed sync handler, passing it the kind from the key string
		switch kind {
		case "namespace":
			err := c.namespaceSyncHandler(key)
			return c.handleErr(err, key)
		case "serviceaccount":
			err := c.serviceAccountSyncHandler(key)
			return c.handleErr(err, key)
		}

		// Finally, if no error occurs we forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		log.Debugf("%s: Successfully synced '%s'", controllerName, key)
		return nil
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
func (c *ContainershipController) handleErr(err error, key interface{}) error {
	if err == nil {
		c.workqueue.Forget(key)
		return nil
	}

	if c.workqueue.NumRequeues(key) < maxRetriesContrainershipController {
		c.workqueue.AddRateLimited(key)
		return fmt.Errorf("error syncing '%v': %s. has been resynced %v times", key, err.Error(), c.workqueue.NumRequeues(key))
	}

	c.workqueue.Forget(key)
	log.Infof("Dropping %q out of the queue: %v", key, err)
	return err
}

// namespaceSyncHandler is put in the queue to be processed on namespace add.
// this allows us to create a containership service account in the namespace
// to be able to manage other resources on the cluster
func (c *ContainershipController) namespaceSyncHandler(key string) error {
	_, _, name, _ := tools.SplitMetaResourceNamespaceKeyFunc(key)

	namespace, err := c.namespacesLister.Get(name)
	if err != nil {
		// If the namespace is not found it was deleted before the workqueue
		// got to processing it and we should ignore it
		if errors.IsNotFound(err) {
			return nil
		}

		return err
	}

	if namespace.Status.Phase == corev1.NamespaceTerminating {
		return nil
	}

	c.recorder.Event(namespace, corev1.EventTypeNormal, "CreateServiceAccount",
		"Detected namespace change")

	_, err = c.kubeclientset.CoreV1().ServiceAccounts(name).Create(newServiceAccount(name))

	// If the error is that the SA already exists, we want to clear the
	// error so that it will be ignored
	if err != nil && errors.IsAlreadyExists(err) == false {
		c.recorder.Eventf(namespace, corev1.EventTypeWarning, "CreateServiceAccountError",
			"Error creating ServiceAccount: %s", err.Error())
		return err
	}

	return nil
}

// service account is only queued when a containership managed service account
// is deleted and the namespace is not being deleted. This will then create a
// service account in the namespace with the containership managed label
func (c *ContainershipController) serviceAccountSyncHandler(key string) error {
	_, nsName, _, _ := tools.SplitMetaResourceNamespaceKeyFunc(key)

	_, err := c.kubeclientset.CoreV1().
		ServiceAccounts(nsName).
		Get(constants.ContainershipServiceAccountName, metav1.GetOptions{})
	// If there is no error that means it already exists and we don't need to do
	// anything
	if err == nil {
		return nil
	}

	// We only explicitly need the namespace for recording, but if we can't get it
	// then that's still a problem.
	// TODO this breaks the convention of only recording on the object that the
	// syncHandler concerns (we're recording on the Namespace here, not the SA
	// as one would expect). Maybe we can do this better.
	ns, err := c.namespacesLister.Get(nsName)
	if err != nil {
		return err
	}

	if errors.IsNotFound(err) {
		c.recorder.Event(ns, corev1.EventTypeNormal, "RecreateServiceAccount",
			"Detected namespace deletion")
		_, err = c.kubeclientset.CoreV1().
			ServiceAccounts(nsName).
			Create(newServiceAccount(constants.ContainershipServiceAccountName))
	}
	if err != nil {
		c.recorder.Eventf(ns, corev1.EventTypeWarning, "RecreateServiceAccountError",
			"Error recreating ServiceAccount: %s", err.Error())
	}

	return err
}

// enqueueNamespace takes a Namespace resource and converts it into a kind/namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Namespace.
func (c *ContainershipController) enqueueNamespace(obj interface{}) {
	key, err := tools.MetaResourceNamespaceKeyFunc("namespace", obj)
	if err != nil {
		log.Error(err)
		return
	}
	c.workqueue.AddRateLimited(key)
}

// enqueueServiceAccount takes a ServiceAccount resource and converts it into a kind/namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Service Account.
func (c *ContainershipController) enqueueServiceAccount(obj interface{}) {
	key, err := tools.MetaResourceNamespaceKeyFunc("serviceaccount", obj)
	if err != nil {
		log.Error(err)
		return
	}
	c.workqueue.AddRateLimited(key)
}

// newServiceAccount creates a new Service Account for a Namespace
func newServiceAccount(namespace string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.ContainershipServiceAccountName,
			Namespace: namespace,
			Labels:    constants.BuildContainershipLabelMap(nil),
		},
	}
}
