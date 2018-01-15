package coordinator

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/containership/cloud-agent/internal/constants"
	"github.com/containership/cloud-agent/internal/log"
	"github.com/containership/cloud-agent/internal/tools"

	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	corelistersv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

const (
	// Type of agent that runs this controller
	controllerName = "containership"
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
func NewContainershipController(
	kubeclientset kubernetes.Interface,
	kubeInformerFactory kubeinformers.SharedInformerFactory) *ContainershipController {

	// Instantiate resource informers
	namespaceInformer := kubeInformerFactory.Core().V1().Namespaces()
	serviceAccountInformer := kubeInformerFactory.Core().V1().ServiceAccounts()

	log.Info(controllerName, ": Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(log.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerName})

	controller := &ContainershipController{
		kubeclientset: kubeclientset,

		// Listers are used for cache inspection and Synced functions
		// are used to wait for cache synchronization
		namespacesLister: namespaceInformer.Lister(),
		namespacesSynced: namespaceInformer.Informer().HasSynced,

		serviceAccountsLister: serviceAccountInformer.Lister(),
		serviceAccountsSynced: serviceAccountInformer.Informer().HasSynced,

		workqueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Containership"),
		recorder:  recorder,
	}

	log.Info(controllerName, ": Setting up event handlers")

	// namespace informer listens for add events an queues the namespace
	namespaceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueNamespace,
	})

	// service account informer listens for the delete event and queues the service account
	serviceAccountInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: func(obj interface{}) {
			if constants.IsContainershipManaged(obj) {
				controller.enqueueServiceAccount(obj)
			}
		},
	})

	return controller
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
// attempt to process it, by calling the registrySyncHandler.
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
			runtime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}

		kind, namespace, name, err := tools.SplitMetaResourceNamespaceKeyFunc(key)
		if err != nil {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workqueue.Forget(obj)
			runtime.HandleError(fmt.Errorf("key is in incorrect format to process %#v", obj))
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
			log.Infof("%s: Namespace '%s' for %s in work queue does not exist\n", controllerName, namespace, kind)
			return nil
		}

		// Run the needed sync handler, passing it the kind from the key string
		switch kind {
		case "namespace":
			if err := c.namespaceSyncHandler(key); err != nil {
				return fmt.Errorf("error syncing '%s': %s", key, err.Error())
			}
		case "serviceaccount":
			if err := c.serviceAccountSyncHandler(key); err != nil {
				return fmt.Errorf("error syncing '%s': %s", key, err.Error())
			}
		}

		// Finally, if no error occurs we forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		log.Debugf("%s: Successfully synced '%s'", controllerName, key)
		return nil
	}(obj)

	if err != nil {
		runtime.HandleError(err)
		return true
	}

	return true
}

// namespaceSyncHandler is put in the queue to be processed on namespace add.
// this allows us to create a containership service account in the namespace
// to be able to manage other resources on the cluster
func (c *ContainershipController) namespaceSyncHandler(key string) error {
	_, _, name, _ := tools.SplitMetaResourceNamespaceKeyFunc(key)

	namespace, err := c.namespacesLister.Get(name)
	if err != nil {
		// If the namesace is not found it was deleted before the workqueue
		// got to processing it and we should ignore it
		if errors.IsNotFound(err) {
			return nil
		}

		return err
	}

	if namespace.Status.Phase == corev1.NamespaceTerminating {
		return nil
	}

	_, err = c.kubeclientset.CoreV1().ServiceAccounts(name).Create(newServiceAccount(name))

	// If the error is that the secret already exists, we want to clear the
	// error so that it will be ignored
	if err != nil && errors.IsAlreadyExists(err) == false {
		log.Info(controllerName, ": Error: ", err)
		return err
	}

	return nil
}

// service account is only queued when a containership managed service account
// is deleted and the namespace is not being deleted. This will then create a
// service account in the namespace with the containership managed label
func (c *ContainershipController) serviceAccountSyncHandler(key string) error {
	_, namespace, _, _ := tools.SplitMetaResourceNamespaceKeyFunc(key)

	_, err := c.kubeclientset.CoreV1().
		ServiceAccounts(namespace).
		Get(constants.ContainershipServiceAccountName, metav1.GetOptions{})
	// If there is no error that means it already exists and we dont need to do
	// anything
	if err == nil {
		return nil
	}

	if errors.IsNotFound(err) {
		_, err = c.kubeclientset.CoreV1().
			ServiceAccounts(namespace).
			Create(newServiceAccount(constants.ContainershipServiceAccountName))
	}

	return err
}

// enqueueNamespace takes a Namespace resource and converts it into a kind/namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Namespace.
func (c *ContainershipController) enqueueNamespace(obj interface{}) {
	key, err := tools.MetaResourceNamespaceKeyFunc("namespace", obj)
	if err != nil {
		runtime.HandleError(err)
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
		runtime.HandleError(err)
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
