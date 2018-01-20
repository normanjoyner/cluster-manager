package coordinator

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/containership/cloud-agent/internal/constants"
	"github.com/containership/cloud-agent/internal/log"
	"github.com/containership/cloud-agent/internal/tools"
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
	registryControllerName = "RegistryController"
)

const (
	// DockerConfigStringFormat is used for Docker tokens, using endpoint, and auth token as password
	DockerConfigStringFormat = `{"%s":{"username":"_json_key","password":"%s","email":"none"}}`
	// DockerJSONStringFormat is used for JSON tokens, endpoint is used under auths,
	// while auth is the token generated
	DockerJSONStringFormat = `{"auths":{"%s":{"auth":"%s","email":"none"}}}`
)

// RegistryController is the controller implementation for Registry resources
type RegistryController struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// clientset is a clientset for our own API group
	clientset csclientset.Interface

	namespacesLister corelistersv1.NamespaceLister
	namespacesSynced cache.InformerSynced

	serviceAccountsLister corelistersv1.ServiceAccountLister
	serviceAccountsSynced cache.InformerSynced

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

// NewRegistryController returns a new coordinator controller which watches
// namespace, service accounts, secrets and registries. It's job is to ensure
// each namespace has a secret for each registry, as well as ensuring that the
// containership SA in each namespace contains each secret as an image pull secret
func NewRegistryController(
	kubeclientset kubernetes.Interface,
	clientset csclientset.Interface,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	csInformerFactory csinformers.SharedInformerFactory) *RegistryController {

	// Instantiate resource informers we care about from the factory so they all
	// share the same underlying cache
	namespaceInformer := kubeInformerFactory.Core().V1().Namespaces()
	secretInformer := kubeInformerFactory.Core().V1().Secrets()
	serviceAccountInformer := kubeInformerFactory.Core().V1().ServiceAccounts()

	// informers for containership resources
	registryInformer := csInformerFactory.Containership().V3().Registries()

	// Create event broadcaster
	// Add coordinator types to the default Kubernetes Scheme so Events can be
	// logged for coordinator types.
	csscheme.AddToScheme(scheme.Scheme)
	log.Info(registryControllerName, ": Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(log.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: registryControllerName})

	controller := &RegistryController{
		kubeclientset: kubeclientset,
		clientset:     clientset,

		// Listers are used for cache inspection and Synced functions
		// are used to wait for cache synchronization
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

	log.Info(registryControllerName + ": Setting up event handlers")
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
	// by checking against it's pseudo parent, Registry. Secrets are owned by
	// Registries, but it's not possible to use an actual OwnerRef because a
	// Secret may be in a different namespace than the owning Registry
	secretInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(old, new interface{}) {
			newS := new.(*corev1.Secret)
			oldS := old.(*corev1.Secret)
			if newS.ResourceVersion == oldS.ResourceVersion {
				// Periodic resync will send update events for all known Secrets.
				// Two different versions of the same Secret will always have different RVs.
				return
			}

			if constants.IsContainershipManaged(old) {
				controller.queueSecretOwnerRegistryIfApplicable(new)
			}

		},
		DeleteFunc: func(obj interface{}) {
			if constants.IsContainershipManaged(obj) {
				controller.queueSecretOwnerRegistryIfApplicable(obj)
			}
		},
	})

	// namespace informer listen for add so we can create all registry
	// secrets in the namespace and add a containership service Account
	// to make containership magic happen
	namespaceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueNamespace,
	})

	// set up service account listener to check if update or delete was authorized
	// on containership SA
	serviceAccountInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if constants.IsContainershipManaged(obj) {
				controller.enqueueServiceAccount(obj)
			}
		},
		UpdateFunc: func(old, new interface{}) {
			newSA := new.(*corev1.ServiceAccount)
			oldSA := old.(*corev1.ServiceAccount)
			if newSA.ResourceVersion == oldSA.ResourceVersion {
				// Periodic resync will send update events for all known Service Account.
				// Two different versions of the same Service Account will always have different RVs.
				return
			}

			if constants.IsContainershipManaged(old) {
				controller.enqueueServiceAccount(new)
			}
		},
	})

	return controller
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *RegistryController) Run(numWorkers int, stopCh chan struct{}) {
	defer runtime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	log.Info(registryControllerName + ": Starting controller")

	if ok := cache.WaitForCacheSync(
		stopCh,
		c.secretsSynced,
		c.registriesSynced,
		c.serviceAccountsSynced,
		c.namespacesSynced); !ok {
		// If this channel is unable to wait for caches to sync we stop both
		// the containership controller, and the registry controller
		close(stopCh)
		log.Error(registryControllerName, ": failed to wait for caches to sync")
	}

	log.Info(registryControllerName, ": Starting workers")
	// Launch numWorkers amount of workers to process resources
	for i := 0; i < numWorkers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	log.Info(registryControllerName, ": Started workers")
	<-stopCh
	log.Info(registryControllerName, ": Shutting down workers")
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *RegistryController) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the registrySyncHandler.
func (c *RegistryController) processNextWorkItem() bool {
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
			log.Infof("%s: Namespace '%s' for %s in work queue does not exist\n", registryControllerName, namespace, kind)
			return nil
		}

		// Run the needed sync handler, passing it the kind from the key string
		switch kind {
		case "registry":
			// If the registry is not being modified in the containership
			// namespace we don't care about the event (shouldn't happen)
			if namespace != constants.ContainershipNamespace {
				return nil
			}

			if err := c.registrySyncHandler(key); err != nil {
				return fmt.Errorf("error syncing '%s': %s", key, err.Error())
			}
		case "serviceaccount":
			// Check the name of the SA. We only want to modify the one we own in each
			// namespace that is named ContainershipServiceAccountName. If it is not a
			// Service account we own, return nil so it doesn't get added back to the queue
			if name != constants.ContainershipServiceAccountName {
				return nil
			}

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
		log.Debugf("%s: Successfully synced '%s'", registryControllerName, key)
		return nil
	}(obj)

	if err != nil {
		runtime.HandleError(err)
		return true
	}

	return true
}

// check for the status of the namespace, to short circuit acting on resource in
// and deleting namespace
func checkNamespace(namespacesLister corelistersv1.NamespaceLister, n string) (terminatingOrTerminated bool, err error) {
	namespace, err := namespacesLister.Get(n)
	if err != nil {
		if errors.IsNotFound(err) {
			return true, nil
		}

		return false, err
	}

	// Check that we are not creating things in a namespace that is terminating
	if namespace.Status.Phase == corev1.NamespaceTerminating {
		return true, nil
	}

	return false, nil
}

// serviceAccountSyncHandler gets a key from the work queue when a namespace is
// added or a registry is modified. Otherwise, it gets the current registries
// that will have corresponding secrets in the namespace and attaches them to the
// service account as image pull secrets. We return errors for this to be re-queued
// if there is an error getting the needed image pull secrets, or creating/updating
// the service account to the desired state
func (c *RegistryController) serviceAccountSyncHandler(key string) error {
	_, namespace, name, err := tools.SplitMetaResourceNamespaceKeyFunc(key)

	sa, err := c.serviceAccountsLister.ServiceAccounts(namespace).Get(name)
	if err != nil {
		return err
	}

	// get a slice of all containership secrets so they can be added as
	// image pull secrets to the containership service account
	imagePullSecrets, err := c.getUpdatedImagePullSecrets()
	if err != nil {
		return err
	}

	if !areImagePullSecretsEqual(imagePullSecrets, sa.ImagePullSecrets) {
		// You should NEVER modify objects from the store. It's a read-only, local cache.
		// You can use DeepCopy() to make a deep copy of original object and modify
		// the copy, and call Update() so that the cache is never directly mutated
		log.Infof("Image pull secrets have changed for Service Account %s, in namespace %s, overwriting...", name, namespace)
		c.recorder.Event(sa, corev1.EventTypeNormal, "UpdateServiceAccountSecrets",
			"Detected out-of-date image pull secrets")
		saCopy := sa.DeepCopy()
		saCopy.ImagePullSecrets = imagePullSecrets

		_, err = c.kubeclientset.CoreV1().ServiceAccounts(namespace).Update(saCopy)

		if err != nil {
			c.recorder.Eventf(sa, corev1.EventTypeWarning, "UpdateServiceAccountSecretsError",
				"Error updating ServiceAccount secrets: %s", err.Error())
			return err
		}
	}

	return nil
}

// areImagePullSecretsEqual checks to see if two slices of type corev1.LocalObjectReference
// are equal
func areImagePullSecretsEqual(currentSecrets, cachedImagePullSecrets []corev1.LocalObjectReference) bool {
	if len(currentSecrets) != len(cachedImagePullSecrets) {
		return false
	}
	// TODO write UT
	ipsDic := make(map[string]corev1.LocalObjectReference)
	for _, ips := range cachedImagePullSecrets {
		ipsDic[ips.Name] = ips
	}

	for _, ips := range currentSecrets {
		if _, ok := ipsDic[ips.Name]; !ok {
			return false
		}
	}

	return true
}

// getUpdatedImagePullSecrets makes a slice of []corev1.LocalObjectReference
// to be attached to a service account with all the secrets that have been created
// from registries
func (c *RegistryController) getUpdatedImagePullSecrets() ([]corev1.LocalObjectReference, error) {
	// TODO: could make this more performant to not have to regenerate this object
	// for every SA add
	registries, err := c.registriesLister.Registries(constants.ContainershipNamespace).List(labels.NewSelector())
	imagePullSecrets := make([]corev1.LocalObjectReference, 0)
	if err != nil {
		return imagePullSecrets, err
	}

	for _, registry := range registries {
		imagePullSecrets = append(imagePullSecrets, corev1.LocalObjectReference{
			Name: registry.Name,
		})
	}

	return imagePullSecrets, nil
}

func (c *RegistryController) addServiceAccountToWorkqueue(namespace string) {
	c.workqueue.AddRateLimited("serviceaccount/" + namespace + "/" + constants.ContainershipServiceAccountName)
}

// registrySyncHandler compares the actual state with the desired, and attempts to
// converge the two. It is called when a registry is added, updated, or deleted, as
// well as if a secret is modified that belongs to a registry. It then makes sure
// its child secrets in every namespace are in the desired state, and queues the
// containership service account in every namespace to be checked. If there is an
// error getting the namespaces, registry, or modifying secrets this returns
// an error and is re-queued
func (c *RegistryController) registrySyncHandler(key string) error {
	_, namespace, name, err := tools.SplitMetaResourceNamespaceKeyFunc(key)

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
			// If the registry has been deleted we need to go through every
			// namespace and delete the secret that it references. Keeping with their
			// forced parent child relationship, since we can't have them automatically
			// deleted using OwnerRefs. This is because the Owner has to be
			// in the same namespace as its child
			for _, ns := range namespaces {
				// We won't record here because there's no good object to record on
				// Add service account for each namespace to queue so old secrets get
				// removed from ImagePullSecrets
				c.addServiceAccountToWorkqueue(ns.Name)
			}

			runtime.HandleError(fmt.Errorf("Registry '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	var namespacesThatContainRegistry []*corev1.Namespace
	for _, ns := range namespaces {
		log.Debugf("%s: Searching namespace %s, for secret %s", registryControllerName, ns.Name, registry.Name)
		_, err = c.secretsLister.Secrets(ns.Name).Get(registry.Name)

		// If the secret doesn't exist, we'll create it, if there was
		// any other kind of error we break so that error can be returned
		// and the registry can be reprocessed
		if errors.IsNotFound(err) {
			c.recorder.Eventf(registry, corev1.EventTypeNormal, "CreateSecret",
				"Detected missing secret in namespace %s, creating", ns.Name)

			_, err = c.kubeclientset.CoreV1().Secrets(ns.Name).Create(newSecret(registry))

			// Add service account for each namespace to queue so newly added secrets
			// get their ID added to ImagePullSecrets
			c.addServiceAccountToWorkqueue(ns.Name)
		} else if err != nil {
			// If an error occurs during Get/Create, we'll requeue the item so we can
			// attempt processing again later. This could have been caused by a
			// temporary network failure, or any other transient reason.
			c.recorder.Eventf(registry, corev1.EventTypeWarning, "ListSecretError",
				"Error listing secrets in namespace %s: %s", ns.Name, err.Error())
			return err
		} else {
			// keep a list of namespaces that already contain the secret
			// so that we can iterate on them for updating
			namespacesThatContainRegistry = append(namespacesThatContainRegistry, ns)
		}
	}

	// TODO: according to the Sync() spec we should only be getting an update
	// registry requests if the registry has changed and no longer equals the secret
	// shouldn't need to compare here but double check/make sure that is implemented correctly
	for _, ns := range namespacesThatContainRegistry {
		c.recorder.Eventf(registry, corev1.EventTypeNormal, "UpdateSecret",
			"Updating secret in namespace %s", ns.Name)
		_, err = c.kubeclientset.CoreV1().Secrets(ns.Name).Update(newSecret(registry))

		if err != nil {
			c.recorder.Eventf(registry, corev1.EventTypeWarning, "UpdateSecretError",
				"Error updating secret in namespace %s: %s", ns.Name, err.Error())
			return err
		}
	}

	return nil
}

// namespaceSyncHandler is put in the queue to be processed on namespace add.
// this lets us know when a new namespace is processed so we can add all the
// current registries as secrets to the namespace.
func (c *RegistryController) namespaceSyncHandler(key string) error {
	_, _, nsName, err := tools.SplitMetaResourceNamespaceKeyFunc(key)

	registries, err := c.registriesLister.Registries(constants.ContainershipNamespace).List(labels.NewSelector())
	if err != nil {
		return err
	}

	// We only explicitly need the namespace for recording, but if we can't get it
	// then that's still a problem.
	ns, err := c.namespacesLister.Get(nsName)
	if err != nil {
		return err
	}

	for _, registry := range registries {
		_, err = c.kubeclientset.CoreV1().Secrets(nsName).Create(newSecret(registry))

		// If the error is that the secret already exists, we want to clear the
		// error so that it will be ignored
		if errors.IsAlreadyExists(err) == true {
			err = nil
		}

		// If there is an error creating the secret, and the error
		// is not that the Secret already exists we want to
		// requeue this namespace to be reprocessed so all secrets are created
		if err != nil {
			c.recorder.Eventf(ns, corev1.EventTypeWarning, "UpdateNamespaceSecretsError",
				"Error updating secrets in namespace: %s", err.Error())
			break
		}
	}

	return err
}

// enqueueRegistry takes a Registry resource and converts it into a kind/namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Registry.
func (c *RegistryController) enqueueRegistry(obj interface{}) {
	key, err := tools.MetaResourceNamespaceKeyFunc("registry", obj)
	if err != nil {
		runtime.HandleError(err)
		return
	}
	c.workqueue.AddRateLimited(key)
}

// enqueueServiceAccount takes a ServiceAccount resource and converts it into a kind/namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than ServiceAccount.
func (c *RegistryController) enqueueServiceAccount(obj interface{}) {
	key, err := tools.MetaResourceNamespaceKeyFunc("serviceaccount", obj)
	if err != nil {
		runtime.HandleError(err)
		return
	}
	c.workqueue.AddRateLimited(key)
}

// enqueueNamespace takes a Namespace resource and converts it into a kind/namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Namespace.
func (c *RegistryController) enqueueNamespace(obj interface{}) {
	key, err := tools.MetaResourceNamespaceKeyFunc("namespace", obj)
	if err != nil {
		runtime.HandleError(err)
		return
	}
	c.workqueue.AddRateLimited(key)
}

// queueSecretOwnerRegistryIfApplicable will take any resource implementing metav1.Object and attempt
// to find the Registry resource that 'owns' it. It does this by looking at the
// objects metadata.Name field to find the appropriate registry associated with it.
// It then enqueues that Registry resource to be processed. If the object does not
// have an appropriate parent Registry, it will be skipped.
func (c *RegistryController) queueSecretOwnerRegistryIfApplicable(obj interface{}) {
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
		log.Infof("%s: Recovered deleted object '%s' from tombstone", registryControllerName, object.GetName())
	}

	log.Debugf("%s: Processing object: %s in %s", registryControllerName, object.GetName(), object.GetNamespace())
	if s, ok := obj.(*corev1.Secret); ok {
		// registry will only ever belong to the containership core namespace
		registry, err := c.registriesLister.Registries(constants.ContainershipNamespace).Get(s.Name)
		if err != nil {
			log.Infof("%s: Secret %s does not belong to any known Registries. %v", registryControllerName, s.Name, err)
			return
		}

		c.enqueueRegistry(registry)
		return
	}
}

// newSecret creates a new Secret for a Registry resource. It sets name
// to be the same as its parent registry to couple them together
func newSecret(registry *containershipv3.Registry) *corev1.Secret {
	rs := registry.Spec
	rdt := rs.AuthToken.Type

	labels := constants.BuildContainershipLabelMap(map[string]string{
		"controller": registry.Name,
	})

	data := make(map[string][]byte, 0)
	// Default template and type to be that of docker config
	template := DockerConfigStringFormat
	t := corev1.SecretTypeDockercfg

	// If the authtoken type is set to dockerconfigjson we want to use the secret
	// type, and template for a JSON auth key
	if rdt == "dockerconfigjson" {
		template = DockerJSONStringFormat
		t = corev1.SecretTypeDockerConfigJson
	}

	// Build the data for the secret, containing the endpoint and token,
	// using the docker template chosen, and set the correct data type
	// according to the authtoken type.
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
