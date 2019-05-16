package coordinator

import (
	"fmt"
	"time"

	"github.com/pkg/errors"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"

	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	rbaclistersv1 "k8s.io/client-go/listers/rbac/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	"github.com/containership/cluster-manager/pkg/constants"
	"github.com/containership/cluster-manager/pkg/log"
	"github.com/containership/cluster-manager/pkg/tools"

	csauthv3 "github.com/containership/cluster-manager/pkg/apis/auth.containership.io/v3"
	csclientset "github.com/containership/cluster-manager/pkg/client/clientset/versioned"
	csinformers "github.com/containership/cluster-manager/pkg/client/informers/externalversions"
	csauthlisters "github.com/containership/cluster-manager/pkg/client/listers/auth.containership.io/v3"
)

const (
	authorizationRoleControllerName = "AuthorizationRoleController"

	clusterRoleDelayBetweenRetries = 30 * time.Second

	maxAuthorizationRoleControllerRetries = 10
)

// AuthorizationRoleController syncs Containership roles to Kubernetes roles
type AuthorizationRoleController struct {
	kubeclientset kubernetes.Interface
	csclientset   csclientset.Interface

	clusterRoleLister  rbaclistersv1.ClusterRoleLister
	clusterRolesSynced cache.InformerSynced

	authorizationRoleLister  csauthlisters.AuthorizationRoleLister
	authorizationRolesSynced cache.InformerSynced

	workqueue workqueue.RateLimitingInterface
	recorder  record.EventRecorder
}

// NewAuthorizationRoleController returns a new clusterRole controller
func NewAuthorizationRoleController(kubeclientset kubernetes.Interface,
	clientset csclientset.Interface,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	csInformerFactory csinformers.SharedInformerFactory) *AuthorizationRoleController {
	rateLimiter := workqueue.NewItemExponentialFailureRateLimiter(clusterRoleDelayBetweenRetries, clusterRoleDelayBetweenRetries)

	c := &AuthorizationRoleController{
		kubeclientset: kubeclientset,
		csclientset:   clientset,
		workqueue:     workqueue.NewNamedRateLimitingQueue(rateLimiter, authorizationRoleControllerName),
		recorder:      tools.CreateAndStartRecorder(kubeclientset, authorizationRoleControllerName),
	}

	// Instantiate resource informers
	clusterRoleInformer := kubeInformerFactory.Rbac().V1().ClusterRoles()
	authorizationRoleInformer := csInformerFactory.ContainershipAuth().V3().AuthorizationRoles()

	log.Info(authorizationRoleControllerName, ": Setting up event handlers")

	authorizationRoleInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueueAuthorizationRole,
		UpdateFunc: func(old, new interface{}) {
			// Do not check ResourceVersions. We'll periodically resync to handle e.g.
			// unexpected ClusterRole deletions.
			c.enqueueAuthorizationRole(new)
		},
		// We don't care about deletes because GC will take care of orphaned ClusterRoles
	})

	// Listers are used for cache inspection and Synced functions
	// are used to wait for cache synchronization
	c.clusterRoleLister = clusterRoleInformer.Lister()
	c.clusterRolesSynced = clusterRoleInformer.Informer().HasSynced

	c.authorizationRoleLister = authorizationRoleInformer.Lister()
	c.authorizationRolesSynced = authorizationRoleInformer.Informer().HasSynced

	return c
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *AuthorizationRoleController) Run(numWorkers int, stopCh chan struct{}) {
	defer runtime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	log.Info(authorizationRoleControllerName, ": Starting controller")

	if ok := cache.WaitForCacheSync(
		stopCh,
		c.clusterRolesSynced,
		c.authorizationRolesSynced); !ok {
		// If this channel is unable to wait for caches to sync we stop both
		// the containership controller, and the clusterRole controller
		close(stopCh)
		log.Error("failed to wait for caches to sync")
	}

	log.Info(authorizationRoleControllerName, ": Starting workers")
	// Launch numWorkers amount of workers to process resources
	for i := 0; i < numWorkers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	log.Info(authorizationRoleControllerName, ": Started workers")
	<-stopCh
	log.Info(authorizationRoleControllerName, ": Shutting down workers")
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *AuthorizationRoleController) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem continually pops items off of the workqueue and handles
// them
func (c *AuthorizationRoleController) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		if key, ok = obj.(string); !ok {
			c.workqueue.Forget(obj)
			log.Errorf("expected string in workqueue but got %#v", obj)
			return nil
		}

		err := c.authorizationRoleSyncHandler(key)
		return c.handleErr(err, key)
	}(obj)

	if err != nil {
		log.Error(err)
		return true
	}

	return true
}

func (c *AuthorizationRoleController) handleErr(err error, key interface{}) error {
	if err == nil {
		c.workqueue.Forget(key)
		return nil
	}

	if c.workqueue.NumRequeues(key) < maxAuthorizationRoleControllerRetries {
		c.workqueue.AddRateLimited(key)
		return fmt.Errorf("error syncing %q: %s. Has been resynced %v times", key, err.Error(), c.workqueue.NumRequeues(key))
	}

	c.workqueue.Forget(key)
	log.Infof("Dropping %q out of the queue: %v", key, err)
	return err
}

func (c *AuthorizationRoleController) enqueueAuthorizationRole(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Error(err)
		return
	}

	c.workqueue.AddRateLimited(key)
}

func (c *AuthorizationRoleController) authorizationRoleSyncHandler(key string) error {
	namespace, name, _ := cache.SplitMetaNamespaceKey(key)

	if namespace != constants.ContainershipNamespace {
		log.Debugf("Ignoring AuthorizationRole %s in namespace %s", name, namespace)
		return nil
	}

	authorizationRole, err := c.authorizationRoleLister.AuthorizationRoles(namespace).Get(name)
	if err != nil {
		if kubeerrors.IsNotFound(err) {
			// ClusterRole is no longer around so nothing to do
			// The ownerRef garbage collection will take care of the associated ClusterRole
			return nil
		}

		return errors.Wrapf(err, "getting AuthorizationRole %s for reconciliation", name)
	}

	id := authorizationRole.Spec.ID
	_, err = c.clusterRoleLister.Get(id)
	if err != nil {
		if kubeerrors.IsNotFound(err) {
			// We don't have a matching ClusterRole, so create it
			log.Infof("%s: Creating missing ClusterRole %s", authorizationRoleControllerName, id)
			role := clusterRoleFromAuthorizationRole(*authorizationRole)
			_, err = c.kubeclientset.RbacV1().ClusterRoles().Create(&role)
			return err
		}

		return errors.Wrapf(err, "getting ClusterRole %s for reconciliation", id)
	}

	log.Debugf("%s: Updating existing ClusterRole %s", authorizationRoleControllerName, id)
	role := clusterRoleFromAuthorizationRole(*authorizationRole)
	_, err = c.kubeclientset.RbacV1().ClusterRoles().Update(&role)
	return err
}

func clusterRoleFromAuthorizationRole(authRole csauthv3.AuthorizationRole) rbacv1.ClusterRole {
	rules := make([]rbacv1.PolicyRule, len(authRole.Spec.Rules))
	for i, authRule := range authRole.Spec.Rules {
		rules[i] = rbacv1.PolicyRule{
			Verbs:           authRule.Verbs,
			APIGroups:       authRule.APIGroups,
			Resources:       authRule.Resources,
			ResourceNames:   authRule.ResourceNames,
			NonResourceURLs: authRule.NonResourceURLs,
		}
	}

	return rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:   authRole.Spec.ID,
			Labels: constants.BuildContainershipLabelMap(nil),
			OwnerReferences: []metav1.OwnerReference{
				ownerReferenceForAuthorizationRole(authRole),
			},
		},
		Rules: rules,
	}
}

func ownerReferenceForAuthorizationRole(authRole csauthv3.AuthorizationRole) metav1.OwnerReference {
	return metav1.OwnerReference{
		// Can't use authRole TypeMeta because it's not guaranteed to be filled in
		APIVersion: csauthv3.SchemeGroupVersion.String(),
		Kind:       "AuthorizationRole",
		Name:       authRole.Name,
		UID:        authRole.UID,
	}
}
