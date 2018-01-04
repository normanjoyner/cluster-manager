package coordinator

import (
	"fmt"
	"log"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"

	containershipv3 "github.com/containership/cloud-agent/pkg/apis/containership.io/v3"
	csclientset "github.com/containership/cloud-agent/pkg/client/clientset/versioned"
	csscheme "github.com/containership/cloud-agent/pkg/client/clientset/versioned/scheme"
	csinformers "github.com/containership/cloud-agent/pkg/client/informers/externalversions"
	cslisters "github.com/containership/cloud-agent/pkg/client/listers/containership.io/v3"

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
	controllerAgentName = "coordinator"
	// ContainershipNamespace is that namespace in which all containership
	// resources will live
	ContainershipNamespace = "containership-core"
	// ContainershipServiceAccountName is the name of containership controlled
	// service account in every namespace
	ContainershipServiceAccountName = "containership"
	// SuccessSynced is used as part of the Event 'reason' when a Registry is synced
	SuccessSynced = "Synced"
	// MessageResourceSynced is the message used for an Event fired when a Registry
	// is synced successfully
	MessageResourceSynced = "%q synced successfully"
	// DockerConfigTemp is used for Docker tokens, using endpoint, and auth token as password
	DockerConfigTemp = `{"%s":{"username":"oauth2accesstoken","password":"%s","email":"none"}}`
	// DockerJSONTemplate is used for JSON tokens, endpoint is used under auths,
	// while auth is the token generated
	DockerJSONTemplate = `{"auths":{"%s":{"auth":"%s","email":"none"}}}`
)

// Controller is the controller implementation for Registry resources
type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// clientset is a clientset for our own API group
	clientset csclientset.Interface

	namespacesLister      corelistersv1.NamespaceLister
	namespacesSynced      cache.InformerSynced
	serviceAccountsLister corelistersv1.ServiceAccountLister
	serviceAccountsSynced cache.InformerSynced
	registriesLister      cslisters.RegistryLister
	registriesSynced      cache.InformerSynced
	secretsLister         corelistersv1.SecretLister
	secretsSynced         cache.InformerSynced

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

	// obtain references to shared index informers for the Namespaces, Secrets,
	// Service Accounts and Registry types.
	namespaceInformer := kubeInformerFactory.Core().V1().Namespaces()
	secretInformer := kubeInformerFactory.Core().V1().Secrets()
	serviceAccountInformer := kubeInformerFactory.Core().V1().ServiceAccounts()

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

		namespacesLister: namespaceInformer.Lister(),
		namespacesSynced: namespaceInformer.Informer().HasSynced,

		serviceAccountsLister: serviceAccountInformer.Lister(),
		serviceAccountsSynced: serviceAccountInformer.Informer().HasSynced,

		registriesLister: registryInformer.Lister(),
		registriesSynced: registryInformer.Informer().HasSynced,

		secretsLister: secretInformer.Lister(),
		secretsSynced: secretInformer.Informer().HasSynced,

		workqueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Registries"),
		recorder:  recorder,
	}

	log.Println("Setting up event handlers")
	// set up an event handler for when there is any change to a Registry resources
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

	// secret informer to check if the update, or deletion was authorized
	// by checking against it's sudo parent, Registry
	secretInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(old, new interface{}) {
			newS := new.(*corev1.Secret)
			oldS := old.(*corev1.Secret)
			if newS.ResourceVersion == oldS.ResourceVersion {
				// Periodic resync will send update events for all known Secrets.
				// Two different versions of the same Secret will always have different RVs.
				return
			}
			controller.handleObject(new)
		},
		DeleteFunc: controller.handleObject,
	})

	// namespace informer listen for add so we can create all registry
	// secrets in the namespace and add a containership service Account
	// to make containership magic happen
	namespaceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueNamespace,
	})

	// set up service account listener to check if update or delete was authorized
	// on containership sa account
	serviceAccountInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(old, new interface{}) {
			newSA := new.(*corev1.ServiceAccount)
			oldSA := old.(*corev1.ServiceAccount)
			if newSA.ResourceVersion == oldSA.ResourceVersion {
				// Periodic resync will send update events for all known Service Account.
				// Two different versions of the same Service Account will always have different RVs.
				return
			}
			controller.enqueueServiceAccount(new)
		},
		DeleteFunc: controller.enqueueServiceAccount,
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

	if ok := cache.WaitForCacheSync(stopCh, c.secretsSynced, c.registriesSynced, c.serviceAccountsSynced, c.namespacesSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	log.Println("Starting workers")
	// Launch threadiness amount of workers to process resources
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	log.Println("Started workers")
	<-stopCh
	log.Println("Shutting down workers")

	return nil
}

// MetaResourceNamespaceKeyFunc is a convenient KeyFunc which knows how to make
// keys for API objects which implement meta.Interface.
// The key uses the format <kind>/<namespace>/<name> unless <namespace> is empty,
// then it's <kind>/<name>.
func MetaResourceNamespaceKeyFunc(kind string, obj interface{}) (string, error) {
	if key, ok := obj.(string); ok {
		return string(key), nil
	}
	m, err := meta.Accessor(obj)

	if err != nil {
		return "", fmt.Errorf("object has no meta: %v", err)
	}

	if len(m.GetNamespace()) > 0 {
		return kind + "/" + m.GetNamespace() + "/" + m.GetName(), nil
	}
	return kind + "/" + m.GetName(), nil
}

// SplitMetaResourceNamespaceKeyFunc returns the kind, namespace and name that
// MetaResourceNamespaceKeyFunc encoded into key.
func SplitMetaResourceNamespaceKeyFunc(key string) (kind, namespace, name string, err error) {
	parts := strings.Split(key, "/")
	switch len(parts) {
	case 2:
		// kind and name only
		return parts[0], "", parts[1], nil
	case 3:
		// kind, namespace and name
		return parts[0], parts[1], parts[2], nil
	}

	return "", "", "", fmt.Errorf("unexpected key format: %q", key)
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

		kind, namespace, name, err := SplitMetaResourceNamespaceKeyFunc(key)

		if err != nil {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workqueue.Forget(obj)
			runtime.HandleError(fmt.Errorf("key is in incorrect format to process %#v", obj))
			return nil
		}

		if namespace == "" && name != "" {
			namespace = name
		}

		terminatingOrTerminated, err := c.checkNamespace(namespace)
		if err != nil {
			return err
		}

		if terminatingOrTerminated == true {
			log.Printf("Namespace '%s' for %s in work queue does not exists \n", namespace, kind)
			return nil
		}

		// Run the needed sync handler, passing it the kind from they key string
		switch kind {
		case "registry":
			if err := c.registrySyncHandler(key); err != nil {
				return fmt.Errorf("error syncing '%s': %s", key, err.Error())
			}
		case "serviceaccount":
			if err := c.serviceAccountSyncHandler(key); err != nil {
				return fmt.Errorf("error syncing '%s': %s", key, err.Error())
			}
		case "namespace":
			if err := c.namespaceSyncHandler(key); err != nil {
				return fmt.Errorf("error syncing '%s': %s", key, err.Error())
			}
		}
		// Finally, if no error occurs we forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		log.Printf("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		runtime.HandleError(err)
		return true
	}

	return true
}

// check for the namespace to know if it should be acted on or not
func (c *Controller) checkNamespace(n string) (terminatingOrTerminated bool, err error) {
	// Make sure the namespace that the service account resides in still exists
	namespace, err := c.namespacesLister.Get(n)
	if err != nil {
		if errors.IsNotFound(err) {
			return true, nil
		}

		return false, err
	}

	// Check that we are not creating things in a namespace that is terminating
	if namespace.Status.Phase == "Terminating" {
		return true, nil
	}

	return false, nil
}

func (c *Controller) serviceAccountSyncHandler(key string) error {
	_, namespace, name, err := SplitMetaResourceNamespaceKeyFunc(key)

	// Check the name of the SA. We only want to modify the one we own in each
	// namespace that is named ContainershipServiceAccountName. If it is not a
	// Service account we own, return nil so it doesn't get added back to the queue
	if name != ContainershipServiceAccountName {
		return nil
	}

	// get an array of all containership secrets so they can be add as
	// image pull secrets to the containership service account
	imagePullSecrets, err := c.GetUpdatedImagePullSecrets()

	if err != nil {
		return err
	}

	sa, err := c.serviceAccountsLister.ServiceAccounts(namespace).Get(name)
	if err != nil {
		// If the contianership service account is missing create it
		if errors.IsNotFound(err) {
			_, err = c.kubeclientset.CoreV1().ServiceAccounts(namespace).Create(newServiceAccount(namespace, imagePullSecrets))
			// Return the status of kubernetes apis create,
			// if it is not nil the service account will be requeued, otherwise its
			// good to go
			return err
		}

		return err
	}

	if err != nil {
		return err
	}

	saCopy := sa.DeepCopy()
	saCopy.ImagePullSecrets = imagePullSecrets

	_, err = c.kubeclientset.CoreV1().ServiceAccounts(namespace).Update(saCopy)

	if err != nil {
		return err
	}

	return nil
}

// getUpdatedImagePullSecrets makes an array of []corev1.LocalObjectReference
// to be attacked to a service account with all the secrets that have been created
// from registries
func (c *Controller) getUpdatedImagePullSecrets() ([]corev1.LocalObjectReference, error) {
	// TODO: could make this more performat to not have to regenerate this object
	// for every SA add
	registries, err := c.registriesLister.Registries(ContainershipNamespace).List(labels.NewSelector())
	imagePullSecrets := make([]corev1.LocalObjectReference, 0)
	if err != nil {
		return imagePullSecrets, err
	}

	for _, registry := range registries {
		imagePullSecrets = append(imagePullSecrets, corev1.LocalObjectReference{registry.Name})
	}

	return imagePullSecrets, nil
}

func (c *Controller) namespaceSyncHandler(key string) error {
	_, _, name, err := SplitMetaResourceNamespaceKeyFunc(key)

	registries, err := c.registriesLister.Registries(ContainershipNamespace).List(labels.NewSelector())

	if err != nil {
		return err
	}

	for _, registry := range registries {
		_, err = c.kubeclientset.CoreV1().Secrets(name).Create(newSecret(registry))

		// If the error is that the secret already exists, we want to clear the
		// error so that it will be ignore it
		if errors.IsAlreadyExists(err) == true {
			err = nil
		}

		// If there is an error creating the secret, and the error
		// is not that the Secret already exists we want to
		// requeue this namespace to be reprocessed so all secrets are created
		if err != nil {
			break
		}
	}

	if err != nil {
		return err
	}

	// TODO is this bad practice :/
	c.workqueue.AddRateLimited("serviceaccount/" + name + "/" + ContainershipServiceAccountName)

	return nil
}

// registrySyncHandler compares the actual state with the desired, and attempts to
// converge the two. By creating needed secrets and enqueuing a Service Account
// for each namespace
func (c *Controller) registrySyncHandler(key string) error {
	_, namespace, name, err := SplitMetaResourceNamespaceKeyFunc(key)
	if err != nil || namespace != ContainershipNamespace {
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

				// Add service account for each namespace to queue so old secrets get
				// removed from ImagePullSecrets
				c.workqueue.AddRateLimited("serviceaccount/" + n.Name + "/" + ContainershipServiceAccountName)
			}
			runtime.HandleError(fmt.Errorf("Registry '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	var registrySecretExists []*corev1.Namespace
	for _, n := range namespaces {
		log.Printf("Searching namespace %s, for secret %s", n.Name, name)
		_, err = c.secretsLister.Secrets(n.Name).Get(name)

		// If the secret doesn't exist, we'll create it, if there was
		// any other kind of error we break so that error can be returned
		// and the registry can be reprocessed
		if errors.IsNotFound(err) {
			_, err = c.kubeclientset.CoreV1().Secrets(n.Name).Create(newSecret(registry))
		} else if err != nil {
			break
		} else {
			// keep a list of namespaces that already contain the secret
			// so that we can iterate on them for updating
			registrySecretExists = append(registrySecretExists, n)
		}
	}

	// If an error occurs during Get/Create, we'll requeue the item so we can
	// attempt processing again later. This could have been caused by a
	// temporary network failure, or any other transient reason.
	if err != nil {
		return err
	}

	// TODO: according to the Sync() spec we should only be getting an update
	// registry requests if the registry has changed and no longer equals the secret
	// shouldn't need to compare here but double check/make sure that is implemented correctly
	for _, n := range registrySecretExists {
		//var secret *corev1.Secret
		log.Printf("Updating secret %s in namespace %s", registry.Name, n.Name)
		_, err = c.kubeclientset.CoreV1().Secrets(n.Name).Update(newSecret(registry))

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

	for _, n := range namespaces {
		c.workqueue.AddRateLimited("serviceaccount/" + n.Name + "/" + ContainershipServiceAccountName)
	}

	c.recorder.Event(registry, corev1.EventTypeNormal, SuccessSynced, fmt.Sprintf(MessageResourceSynced, "Registry"))

	return nil
}

// enqueueRegistry takes a Registry resource and converts it into a kind/namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Registry.
func (c *Controller) enqueueRegistry(obj interface{}) {
	var key string
	var err error
	if key, err = MetaResourceNamespaceKeyFunc("registry", obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.workqueue.AddRateLimited(key)
}

// enqueueNamespace takes a Namespace resource and converts it into a kind/namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Namespace.
func (c *Controller) enqueueNamespace(obj interface{}) {
	var key string
	var err error
	if key, err = MetaResourceNamespaceKeyFunc("namespace", obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.workqueue.AddRateLimited(key)
}

// enqueueServiceAccount takes a ServiceAccount resource and converts it into a kind/namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than ServiceAccount.
func (c *Controller) enqueueServiceAccount(obj interface{}) {
	var key string
	var err error
	if key, err = MetaResourceNamespaceKeyFunc("serviceaccount", obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.workqueue.AddRateLimited(key)
}

// handleObject will take any resource implementing metav1.Object and attempt
// to find the Registry resource that 'owns' it. It does this by looking at the
// objects metadata.Name field to find the appropriate registry associated with it.
// It then enqueues that Registry resource to be processed. If the object does not
// have an appropriate parent Registry, it will be skipped.
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

		registry, err := c.registriesLister.Registries(ContainershipNamespace).Get(s.Name)
		if err != nil {
			log.Printf("Secret %s does not belong to any known Registryies.", s.Name, err)
			return
		}

		c.enqueueRegistry(registry)
		return
	}
}

// newServiceAccount creates a new Service Account for a Namespace resource if
// the namespace does not currently contain a containership service account
func newServiceAccount(namespace string, imagePullSecrets []corev1.LocalObjectReference) *corev1.ServiceAccount {
	labels := map[string]string{
		"containershio.io": "managed",
	}

	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ContainershipServiceAccountName,
			Namespace: namespace,
			Labels:    labels,
		},
		ImagePullSecrets: imagePullSecrets,
	}
}

// newSecret creates a new Secret for a Registry resource. It also sets
// the appropriate OwnerReferences on the resource so handleObject can discover
// the Registry resource that 'owns' it.
func newSecret(registry *containershipv3.Registry) *corev1.Secret {
	log.Printf("\n Creating new secret definition using registry: %s \n", registry.Name)
	rs := registry.Spec
	rdt := rs.AuthToken.Type

	// Add containership management label for easy filtering
	labels := map[string]string{
		"containershio.io": "managed",
		"controller":       registry.Name,
	}

	data := make(map[string][]byte, 0)
	// Default tempate and type to be that of docker config
	template := DockerConfigTemp
	t := corev1.SecretTypeDockercfg

	// If the authtoken type is set to dockerconfigjson we want to use the secret
	// type, and template for a JSON auth key
	if rdt == "dockerconfigjson" {
		template = DockerJSONTemplate
		t = corev1.SecretTypeDockerConfigJson
	}

	// Encrypt the token, and endpoint using the template docker template choosen
	data[fmt.Sprintf(".%s", rdt)] = []byte(fmt.Sprintf(template, rs.AuthToken.Endpoint, rs.AuthToken.Token))

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:   registry.Name,
			Labels: labels,
		},
		Data: data,
		Type: t,
	}
}
