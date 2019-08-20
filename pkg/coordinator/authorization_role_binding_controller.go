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
		// We don't need to listen for deletes because the finalizer logic
		// depends only on update events.
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
			// Nothing to do. Cleanup of generated {Cluster,}RoleBindings is
			// handled by the finalizer logic below.
			return nil
		}

		return errors.Wrapf(err, "getting AuthorizationRoleBinding %s for reconciliation", name)
	}

	roleNamespace := authorizationRoleBinding.Spec.Namespace
	id := authorizationRoleBinding.Spec.ID

	if !authorizationRoleBinding.DeletionTimestamp.IsZero() {
		// This AuthorizationRoleBinding is marked for deletion. We must delete
		// the dependent/generated {Cluster,}RoleBinding we may have created
		// and remove the finalizer on the CR before Kubernetes will actually
		// delete it. The name of the generated {Cluster,}RoleBinding will be
		// equal to the ID of this AuthorizationRoleBinding.
		// If any errors occur, then return an error before removing the
		// finalizer. We do this to create the guarantee that if an
		// AuthorizationRoleBinding does not exist then no matching
		// {Cluster,}RoleBindings will exist.
		var err error
		if roleNamespace == "" {
			err = c.deleteClusterRoleBindingIfExists(id)
		} else {
			err = c.deleteRoleBindingIfExists(id, roleNamespace)
		}
		if err != nil {
			return errors.Wrapf(err, "cleaning up generated Rolebinding or ClusterRoleBinding %s", name)
		}

		// Either we successfully removed the dependent {Cluster,}RoleBinding or it
		// didn't exist, so now remove the finalizer and let Kubernetes take
		// over.
		bindingCopy := authorizationRoleBinding.DeepCopy()
		bindingCopy.Finalizers = tools.RemoveStringFromSlice(bindingCopy.Finalizers, constants.AuthorizationRoleBindingFinalizerName)

		_, err = c.csclientset.ContainershipAuthV3().AuthorizationRoleBindings(namespace).Update(bindingCopy)
		if err != nil {
			return errors.Wrap(err, "updating AuthorizationRoleBinding with finalizer removed")
		}

		return nil
	}

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

		// Always call update and let Kubernetes figure out if an update is
		// actually needed instead of determining that ourselves
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

		// Always call update and let Kubernetes figure out if an update is
		// actually needed instead of determining that ourselves
		binding := roleBindingFromAuthorizationRoleBinding(*authorizationRoleBinding)
		_, err = c.kubeclientset.RbacV1().RoleBindings(roleNamespace).Update(&binding)
	}

	return err
}

func (c *AuthorizationRoleBindingController) deleteClusterRoleBindingIfExists(name string) error {
	_, err := c.clusterRoleBindingLister.Get(name)
	if err != nil {
		if kubeerrors.IsNotFound(err) {
			return nil
		}

		return err
	}

	// It does exist, so delete it.
	return c.kubeclientset.RbacV1().ClusterRoleBindings().Delete(name, &metav1.DeleteOptions{})
}

func (c *AuthorizationRoleBindingController) deleteRoleBindingIfExists(name string, namespace string) error {
	_, err := c.roleBindingLister.RoleBindings(namespace).Get(name)
	if err != nil {
		if kubeerrors.IsNotFound(err) {
			return nil
		}

		return err
	}

	// It does exist, so delete it.
	return c.kubeclientset.RbacV1().RoleBindings(namespace).Delete(name, &metav1.DeleteOptions{})
}

func clusterRoleBindingFromAuthorizationRoleBinding(authRoleBinding csauthv3.AuthorizationRoleBinding) rbacv1.ClusterRoleBinding {
	// We can't set an OwnerReference pointing to the AuthorizationRoleBinding
	// here because a ClusterRoleBinding is not namespaced while an
	// AuthorizationRoleBinding is.
	return rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   authRoleBinding.Spec.ID,
			Labels: constants.BuildContainershipLabelMap(nil),
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
	// We can't set an OwnerReference pointing to the AuthorizationRoleBinding
	// here because a RoleBinding is not guaranteed to live in the same
	// namespace as the AuthorizationRoleBinding
	return rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      authRoleBinding.Spec.ID,
			Namespace: authRoleBinding.Spec.Namespace,
			Labels:    constants.BuildContainershipLabelMap(nil),
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
