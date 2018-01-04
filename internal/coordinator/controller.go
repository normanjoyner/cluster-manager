package coordinator

import (
	"fmt"
	"log"
	"time"

	appsv1beta2 "k8s.io/api/apps/v1beta2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	//"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	corelistersv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	containershipv3 "github.com/containership/cloud-agent/pkg/apis/containership.io/v3"
	csclientset "github.com/containership/cloud-agent/pkg/client/clientset/versioned"
	csscheme "github.com/containership/cloud-agent/pkg/client/clientset/versioned/scheme"
	csinformers "github.com/containership/cloud-agent/pkg/client/informers/externalversions"
	cslisters "github.com/containership/cloud-agent/pkg/client/listers/containership.io/v3"
)


const (
	controllerAgentName = "coordinator"
	// Namespace in which all containership resources will live
	ContainershipNamespace = "containership-core"
	// SuccessSynced is used as part of the Event 'reason' when a Foo is synced
	SuccessSynced = "Synced"
	// ErrResourceExists is used as part of the Event 'reason' when a Foo fails
	// to sync due to a Deployment of the same name already existing.
	ErrResourceExists = "ErrResourceExists"

	// MessageResourceExists is the message used for Events when a resource
	// fails to sync due to a Deployment already existing
	MessageResourceExists = "Resource %q already exists and is not managed by Foo"
	// MessageResourceSynced is the message used for an Event fired when a Foo
	// is synced successfully
	MessageResourceSynced = "%q synced successfully"

	// Template used for docker tokens
	DockerConfigTemp   = `{"%s":{"username":"oauth2accesstoken","password":"%s","email":"none"}}`
	// Template used for JSON tokens
	DockerJSONTemplate = `{"auths":{"%s":{"auth":"%s","email":"none"}}}`
)

// Controller is the controller implementation for Registry resources
type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// clientset is a clientset for our own API group
	clientset csclientset.Interface

	//deploymentsLister appslisters.DeploymentLister
	//deploymentsSynced cache.InformerSynced
	namespacesLister corelistersv1.NamespaceLister
	registriesLister cslisters.RegistryLister
	registriesSynced cache.InformerSynced
	secretsLister corelistersv1.SecretLister
	secretsSynced cache.InformerSynced

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

// NewController returns a new coordinator controller
func NewController(
	kubeclientset kubernetes.Interface,
	clientset csclientset.Interface,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	csInformerFactory csinformers.SharedInformerFactory) *Controller {

	// obtain references to shared index informers for the Deployment and Registry
	// types.
	deploymentInformer := kubeInformerFactory.Apps().V1beta2().Deployments()
	namespaceInformer := kubeInformerFactory.Core().V1().Namespaces()
	secretInformer := kubeInformerFactory.Core().V1().Secrets()

	// informers for containership resources
	registryInformer := csInformerFactory.Containership().V3().Registries()


	// Create event broadcaster
	// Add coordinator types to the default Kubernetes Scheme so Events can be
	// logged for coordinator types.
	csscheme.AddToScheme(scheme.Scheme)
	log.Println("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(log.Printf)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		kubeclientset: kubeclientset,
		clientset:     clientset,
		//deploymentsLister: deploymentInformer.Lister(),
		//deploymentsSynced: deploymentInformer.Informer().HasSynced,
		namespacesLister: namespaceInformer.Lister(),
		registriesLister: registryInformer.Lister(),
		registriesSynced: registryInformer.Informer().HasSynced,
		secretsLister: secretInformer.Lister(),
		secretsSynced: secretInformer.Informer().HasSynced,
		workqueue:        workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Registries"),
		recorder:         recorder,
	}

	log.Println("Setting up event handlers")
	// Set up an event handler for when Registry resources change
	registryInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueRegistry,
		UpdateFunc: func(old, new interface{}) {
			newReg := new.(*containershipv3.Registry)
			oldReg := old.(*containershipv3.Registry)
			if oldReg.ResourceVersion == newReg.ResourceVersion {
				return
			}
			controller.enqueueRegistry(new)
		},
		DeleteFunc: controller.enqueueRegistry,
	})
	// secret informer to do stuff on secret change
	secretInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		//AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newS := new.(*corev1.Secret)
			oldS := old.(*corev1.Secret)
			if newS.ResourceVersion == oldS.ResourceVersion {
				// Periodic resync will send update events for all known Deployments.
				// Two different versions of the same Deployment will always have different RVs.
				return
			}
			controller.handleObject(new)
		},
		DeleteFunc: controller.handleObject,
	})

	// Set up an event handler for when Deployment resources change. This
	// handler will lookup the owner of the given Deployment, and if it is
	// owned by a Registry resource will enqueue that Registry resource for
	// processing. This way, we don't need to implement custom logic for
	// handling Deployment resources. More info on this pattern:
	// https://github.com/kubernetes/community/blob/8cafef897a22026d42f5e5bb3f104febe7e29830/contributors/devel/controllers.md
	deploymentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newDepl := new.(*appsv1beta2.Deployment)
			oldDepl := old.(*appsv1beta2.Deployment)
			if newDepl.ResourceVersion == oldDepl.ResourceVersion {
				// Periodic resync will send update events for all known Deployments.
				// Two different versions of the same Deployment will always have different RVs.
				return
			}
			controller.handleObject(new)
		},
		DeleteFunc: controller.handleObject,
	})

	return controller
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	log.Println("Starting Registry controller")

	// TODO
	// Wait for the caches to be synced before starting workers
	//log.Println("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.secretsSynced, c.registriesSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	log.Println("Starting workers")
	// Launch two workers to process Registry resources
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	log.Println("Started workers")
	<-stopCh
	log.Println("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the registrySyncHandler.
func (c *Controller) processNextWorkItem() bool {
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
		// form namespace/name. We do this as the delayed nature of the
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
		// Run the registrySyncHandler, passing it the namespace/name string of the
		// Registry resource to be synced.
		if err := c.registrySyncHandler(key); err != nil {
			return fmt.Errorf("error syncing '%s': %s", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		log.Println("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		runtime.HandleError(err)
		return true
	}

	return true
}


// registrySyncHandler compares the actual state with the desired, and attempts to
// TODO make into sentence.. not sure what you where going for here
// with the current status of the resource.
func (c *Controller) registrySyncHandler(key string) error {
	log.Printf("registrySyncHandler: key=%s\n", key)
	//TODO:// thinkabout.. namespace will only ever be containership-core
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	namespaces, err := c.namespacesLister.List(labels.NewSelector())
	if err != nil {
		return err
	}

	// Get the Registry resource with this namespace/name
	registry, err := c.registriesLister.Registries(namespace).Get(name)
	if err != nil {
		// The Registry resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			for _, n := range namespaces {
				log.Println("Deleting secret in namespace", n.Name)
				c.kubeclientset.CoreV1().Secrets(n.Name).Delete(name, &metav1.DeleteOptions{})
			}
			//TODO:
			// Get SA account in each namespaces using Lister
			// 			enqueue ServiceAccount to workqueue to be checked for
			runtime.HandleError(fmt.Errorf("Registry '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	var registrySecretExists []*corev1.Namespace
	for _, n := range namespaces {
		//var secret *corev1.Secret
		log.Printf("search namespace: %s", n.Name)
		_, err = c.secretsLister.Secrets(n.Name).Get(name)
		// If the secret doesn't exist, we'll create it
		if errors.IsNotFound(err) {
			log.Printf("creating secret to use: %+v", registry)
			_, err = c.kubeclientset.CoreV1().Secrets(n.Name).Create(newSecret(registry, n.Name))
		} else if err != nil {
			break
		} else {
			registrySecretExists = append(registrySecretExists, n)
		}
	}

	// If an error occurs during Get/Create, we'll requeue the item so we can
	// attempt processing again later. This could have been caused by a
	// temporary network failure, or any other transient reason.
	if err != nil {
		return err
	}

	// If the Secret is not controlled by this Registry resource, we should log
	// a warning to the event recorder and return
	//TODO:// do we need something like this, or will the informer filter handle all that?
	// if !metav1.IsControlledBy(secret, registry) {
	// 	msg := fmt.Sprintf(MessageResourceExists, secret.Name)
	// 	c.recorder.Event(registry, corev1.EventTypeWarning, ErrResourceExists, msg)
	// 	return fmt.Errorf(msg)
	// }

	// TODO: according to the Sync() spec we should only be getting an update
	// registry requests if the registry has changed and no longer equals the secret
	// shouldn't need to compare here
	log.Printf("updating secret to use: %+#v", registry.Spec)
	for _, n := range registrySecretExists {
		//var secret *corev1.Secret
		log.Printf("updating secret in namespace: %s", n.Name)
		_, err = c.kubeclientset.CoreV1().Secrets(n.Name).Update(newSecret(registry, n.Name))

		if err != nil {
			break
		}
	}

	// If an error occurs during Update, we'll requeue the item so we can
	// attempt processing again later. This could have been caused by a
	// temporary network failure, or any other transient reason.
	if err != nil {
		return err
	}

	// Finally, we update the status block of the Registry resource to reflect the
	// current state of the world
	// TODO:// implement this later to keep track of authtokens or maybe resouce sync will take care of this..
	// err = c.updateFooStatus(foo, deployment)
	// if err != nil {
	// 	return err
	// }

	c.recorder.Event(registry, corev1.EventTypeNormal, SuccessSynced, fmt.Sprintf(MessageResourceSynced, "Registry"))

	return nil
}

// func registryIsEqualToSecret(registry containershipv3.RegistrySpec, secret coreV1.SecretSpec)

// enqueueRegistry takes a Registry resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Registry.
func (c *Controller) enqueueRegistry(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.workqueue.AddRateLimited(key)
}

// handleObject will take any resource implementing metav1.Object and attempt
// to find the Registry resource that 'owns' it. It does this by looking at the
// objects metadata.ownerReferences field for an appropriate OwnerReference.
// It then enqueues that Registry resource to be processed. If the object does not
// have an appropriate OwnerReference, it will simply be skipped.
func (c *Controller) handleObject(obj interface{}) {
	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			runtime.HandleError(fmt.Errorf("error decoding object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			runtime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
			return
		}
		log.Printf("Recovered deleted object '%s' from tombstone", object.GetName())
	}

	log.Printf("Processing object: %s in %s", object.GetName(), object.GetNamespace())
	if s, ok := obj.(*corev1.Secret); ok {
		// registry will only ever belong to the contianership core namespace
		// secretsNamespace = object.GetNamespace()
		registry, err := c.registriesLister.Registries(ContainershipNamespace).Get(s.Name)
		if err != nil {
			log.Printf("ignoring orphaned object '%s' of registry '%s', %v", s.GetSelfLink(), s.Name, err)
			return
		}

		c.enqueueRegistry(registry)
		return
	}


	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {
		// If this object is not owned by a Registry, we should not do anything more
		// with it.
		if ownerRef.Kind != "Registry" {
			return
		}

		// // registry will only ever belong to the contianership core namespace
		// // secretsNamespace = object.GetNamespace()
		// registry, err := c.registriesLister.Registries(ContainershipNamespace).Get(ownerRef.Name)
		// if err != nil {
		// 	log.Printf("ignoring orphaned object '%s' of registry '%s'", object.GetSelfLink(), ownerRef.Name)
		// 	return
		// }
		//
		// c.enqueueRegistry(registry)
		// return
	}
}

// newSecret creates a new Secret for a Registry resource. It also sets
// the appropriate OwnerReferences on the resource so handleObject can discover
// the Registry resource that 'owns' it.
func newSecret(registry *containershipv3.Registry, namespace string) *corev1.Secret {
	fmt.Sprintf("\n Creating New Secret Definition: %+#v \n", registry)
	fmt.Println("SHOULD BE CREATED IN NAMESPACE: ", namespace)
	rs := registry.Spec
	rdt := rs.AuthToken.Type

	labels := map[string]string{
		"containershio.io": "managed",
		"controller": registry.Name,
	}

	data := make(map[string][]byte, 0)
	template := DockerConfigTemp
	if rdt == "dockerconfigjson" {
		template = DockerJSONTemplate
	}

	data[fmt.Sprintf(".%s", rdt)] = []byte(fmt.Sprintf(template, rs.AuthToken.Endpoint, rs.AuthToken.Token))
	//secretType := fmt.Sprint("kubernetes.io/%s", rdt)
	//TODO: going to need to know the type of registry to get the type that the data should be
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      registry.Name,
			//Namespace: namespace,
			// OwnerReferences: []metav1.OwnerReference{
			// 	*metav1.NewControllerRef(registry, schema.GroupVersionKind{
			// 		Group:   containershipv3.SchemeGroupVersion.Group,
			// 		Version: containershipv3.SchemeGroupVersion.Version,
			// 		Kind:    "Registry",
			// 	}),
			// },
			Labels: labels,
		},
		Data: data,
		Type: "kubernetes.io/dockercfg",
	}
}
