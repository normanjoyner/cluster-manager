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
	authorizationRoleBindingControllerName = "AuthorizationRoleBindingController"

	clusterRoleBindingDelayBetweenRetries = 30 * time.Second

	maxAuthorizationRoleBindingControllerRetries = 10
)

// AuthorizationRoleBindingController syncs Containership role bindings to Kubernetes bindings
type AuthorizationRoleBindingController struct {
	kubeclientset kubernetes.Interface
	csclientset   csclientset.Interface

	clusterRoleBindingLister  rbaclistersv1.ClusterRoleBindingLister
	clusterRoleBindingsSynced cache.InformerSynced

	roleBindingLister  rbaclistersv1.RoleBindingLister
	roleBindingsSynced cache.InformerSynced

	authorizationRoleBindingLister  csauthlisters.AuthorizationRoleBindingLister
	authorizationRoleBindingsSynced cache.InformerSynced

	workqueue workqueue.RateLimitingInterface
	recorder  record.EventRecorder
}

// NewAuthorizationRoleBindingController returns a new clusterRoleBinding controller
func NewAuthorizationRoleBindingController(kubeclientset kubernetes.Interface,
	clientset csclientset.Interface,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	csInformerFactory csinformers.SharedInformerFactory) *AuthorizationRoleBindingController {
	rateLimiter := workqueue.NewItemExponentialFailureRateLimiter(clusterRoleBindingDelayBetweenRetries, clusterRoleBindingDelayBetweenRetries)

	c := &AuthorizationRoleBindingController{
		kubeclientset: kubeclientset,
		csclientset:   clientset,
		workqueue:     workqueue.NewNamedRateLimitingQueue(rateLimiter, authorizationRoleBindingControllerName),
		recorder:      tools.CreateAndStartRecorder(kubeclientset, authorizationRoleBindingControllerName),
	}

	// Instantiate resource informers
	clusterRoleBindingInformer := kubeInformerFactory.Rbac().V1().ClusterRoleBindings()
	roleBindingInformer := kubeInformerFactory.Rbac().V1().RoleBindings()
	authorizationRoleBindingInformer := csInformerFactory.ContainershipAuth().V3().AuthorizationRoleBindings()

	log.Info(authorizationRoleBindingControllerName, ": Setting up event handlers")

	authorizationRoleBindingInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueueAuthorizationRoleBinding,
		UpdateFunc: func(old, new interface{}) {
			// Do not check ResourceVersions. We'll periodically resync to handle e.g.
			// unexpected *Binding deletions.
			c.enqueueAuthorizationRoleBinding(new)
		},
		// We don't care about deletes because GC will take care of orphaned ClusterRoleBindings
	})

	// Listers are used for cache inspection and Synced functions
	// are used to wait for cache synchronization
	c.clusterRoleBindingLister = clusterRoleBindingInformer.Lister()
	c.clusterRoleBindingsSynced = clusterRoleBindingInformer.Informer().HasSynced

	c.roleBindingLister = roleBindingInformer.Lister()
	c.roleBindingsSynced = roleBindingInformer.Informer().HasSynced

	c.authorizationRoleBindingLister = authorizationRoleBindingInformer.Lister()
	c.authorizationRoleBindingsSynced = authorizationRoleBindingInformer.Informer().HasSynced

	return c
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *AuthorizationRoleBindingController) Run(numWorkers int, stopCh chan struct{}) {
	defer runtime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	log.Info(authorizationRoleBindingControllerName, ": Starting controller")

	if ok := cache.WaitForCacheSync(
		stopCh,
		c.clusterRoleBindingsSynced,
		c.authorizationRoleBindingsSynced); !ok {
		// If this channel is unable to wait for caches to sync we stop both
		// the containership controller, and the clusterRoleBinding controller
		close(stopCh)
		log.Error("failed to wait for caches to sync")
	}

	log.Info(authorizationRoleBindingControllerName, ": Starting workers")
	// Launch numWorkers amount of workers to process resources
	for i := 0; i < numWorkers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	log.Info(authorizationRoleBindingControllerName, ": Started workers")
	<-stopCh
	log.Info(authorizationRoleBindingControllerName, ": Shutting down workers")
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *AuthorizationRoleBindingController) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem continually pops items off of the workqueue and handles
// them
func (c *AuthorizationRoleBindingController) processNextWorkItem() bool {
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

		err := c.authorizationRoleBindingSyncHandler(key)
		return c.handleErr(err, key)
	}(obj)

	if err != nil {
		log.Error(err)
		return true
	}

	return true
}

func (c *AuthorizationRoleBindingController) handleErr(err error, key interface{}) error {
	if err == nil {
		c.workqueue.Forget(key)
		return nil
	}

	if c.workqueue.NumRequeues(key) < maxAuthorizationRoleBindingControllerRetries {
		c.workqueue.AddRateLimited(key)
		return fmt.Errorf("error syncing %q: %s. Has been resynced %v times", key, err.Error(), c.workqueue.NumRequeues(key))
	}

	c.workqueue.Forget(key)
	log.Infof("Dropping %q out of the queue: %v", key, err)
	return err
}

func (c *AuthorizationRoleBindingController) enqueueAuthorizationRoleBinding(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Error(err)
		return
	}

	c.workqueue.AddRateLimited(key)
}

func (c *AuthorizationRoleBindingController) authorizationRoleBindingSyncHandler(key string) error {
	namespace, name, _ := cache.SplitMetaNamespaceKey(key)

	if namespace != constants.ContainershipNamespace {
		log.Debugf("Ignoring AuthorizationRoleBinding %s in namespace %s", name, namespace)
		return nil
	}

	authorizationRoleBinding, err := c.authorizationRoleBindingLister.AuthorizationRoleBindings(namespace).Get(name)
	if err != nil {
		if kubeerrors.IsNotFound(err) {
			// ClusterRoleBinding is no longer around so nothing to do
			// The ownerRef garbage collection will take care of the associated ClusterRoleBinding
			return nil
		}

		return errors.Wrapf(err, "getting AuthorizationRoleBinding %s for reconciliation", name)
	}

	id := authorizationRoleBinding.Spec.ID

	roleNamespace := authorizationRoleBinding.Spec.Namespace
	if roleNamespace == "" {
		// No namespace, so this should be synced to a ClusterRoleBinding
		_, err = c.clusterRoleBindingLister.Get(id)
		if err != nil {
			if kubeerrors.IsNotFound(err) {
				// We don't have a matching ClusterRoleBinding, so create it
				log.Infof("%s: Creating missing ClusterRoleBinding %s", authorizationRoleBindingControllerName, id)
				binding := clusterRoleBindingFromAuthorizationRoleBinding(*authorizationRoleBinding)
				_, err = c.kubeclientset.RbacV1().ClusterRoleBindings().Create(&binding)
				return err
			}

			return errors.Wrapf(err, "getting ClusterRoleBinding %s for reconciliation", id)
		}

		log.Debugf("%s: Updating existing ClusterRoleBinding %s", authorizationRoleBindingControllerName, id)
		binding := clusterRoleBindingFromAuthorizationRoleBinding(*authorizationRoleBinding)
		_, err = c.kubeclientset.RbacV1().ClusterRoleBindings().Update(&binding)
	} else {
		// There's a namespace, so we should sync to a RoleBinding in the given namespace
		_, err = c.roleBindingLister.RoleBindings(roleNamespace).Get(id)
		if err != nil {
			if kubeerrors.IsNotFound(err) {
				// We don't have a matching RoleBinding, so create it
				log.Infof("%s: Creating missing RoleBinding %s", authorizationRoleBindingControllerName, id)
				binding := roleBindingFromAuthorizationRoleBinding(*authorizationRoleBinding)
				_, err = c.kubeclientset.RbacV1().RoleBindings(roleNamespace).Create(&binding)
				return err
			}

			return errors.Wrapf(err, "getting RoleBinding %s for reconciliation in namespace %s", id, roleNamespace)
		}

		log.Debugf("%s: Updating existing RoleBinding %s", authorizationRoleBindingControllerName, id)
		binding := roleBindingFromAuthorizationRoleBinding(*authorizationRoleBinding)
		_, err = c.kubeclientset.RbacV1().RoleBindings(roleNamespace).Update(&binding)
	}

	return err
}

func clusterRoleBindingFromAuthorizationRoleBinding(authRoleBinding csauthv3.AuthorizationRoleBinding) rbacv1.ClusterRoleBinding {
	return rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   authRoleBinding.Spec.ID,
			Labels: constants.BuildContainershipLabelMap(nil),
			OwnerReferences: []metav1.OwnerReference{
				ownerReferenceForAuthorizationRoleBinding(authRoleBinding),
			},
		},
		Subjects: []rbacv1.Subject{
			subjectFromBinding(authRoleBinding),
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole", // Containership only creates ClusterRoles
			Name:     authRoleBinding.Spec.AuthorizationRoleID,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}
}

func roleBindingFromAuthorizationRoleBinding(authRoleBinding csauthv3.AuthorizationRoleBinding) rbacv1.RoleBinding {
	return rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      authRoleBinding.Spec.ID,
			Namespace: authRoleBinding.Spec.Namespace,
			Labels:    constants.BuildContainershipLabelMap(nil),
			OwnerReferences: []metav1.OwnerReference{
				ownerReferenceForAuthorizationRoleBinding(authRoleBinding),
			},
		},
		Subjects: []rbacv1.Subject{
			subjectFromBinding(authRoleBinding),
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole", // Containership only creates ClusterRoles
			Name:     authRoleBinding.Spec.AuthorizationRoleID,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}
}

func subjectFromBinding(binding csauthv3.AuthorizationRoleBinding) rbacv1.Subject {
	var subject rbacv1.Subject
	switch binding.Spec.Type {
	case csauthv3.AuthorizationRoleBindingTypeTeam:
		subject = rbacv1.Subject{
			Kind: "Group",
			Name: "containership.io/team_id#" + binding.Spec.TeamID,
		}
	case csauthv3.AuthorizationRoleBindingTypeUser:
		subject = rbacv1.Subject{
			Kind: "User",
			Name: "containership.io/user_id#" + binding.Spec.UserID,
		}
	}

	return subject
}

func ownerReferenceForAuthorizationRoleBinding(binding csauthv3.AuthorizationRoleBinding) metav1.OwnerReference {
	return metav1.OwnerReference{
		// Can't use binding TypeMeta because it's not guaranteed to be filled in
		APIVersion: csauthv3.SchemeGroupVersion.String(),
		Kind:       "AuthorizationRole",
		Name:       binding.Name,
		UID:        binding.UID,
	}
}
