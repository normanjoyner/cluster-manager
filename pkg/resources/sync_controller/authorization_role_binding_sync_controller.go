package synccontroller

import (
	"fmt"

	cscloud "github.com/containership/csctl/cloud"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"

	authv3 "github.com/containership/cluster-manager/pkg/apis/auth.containership.io/v3"
	csclientset "github.com/containership/cluster-manager/pkg/client/clientset/versioned"
	csinformers "github.com/containership/cluster-manager/pkg/client/informers/externalversions"
	authlisters "github.com/containership/cluster-manager/pkg/client/listers/auth.containership.io/v3"

	"github.com/containership/cluster-manager/pkg/constants"
	"github.com/containership/cluster-manager/pkg/log"
	"github.com/containership/cluster-manager/pkg/resources"
	"github.com/containership/cluster-manager/pkg/tools"
)

// AuthorizationRoleBindingSyncController is the implementation for syncing AuthorizationRoleBinding CRDs
type AuthorizationRoleBindingSyncController struct {
	*syncController

	lister        authlisters.AuthorizationRoleBindingLister
	cloudResource *resources.CsAuthorizationRoleBindings
}

const (
	authorizationRoleBindingSyncControllerName = "AuthorizationRoleBindingSyncController"
)

// NewAuthorizationRoleBinding returns a AuthorizationRoleBindingSyncController that will be in control of pulling from cloud
// comparing to the CRD cache and modifying based on those compares
func NewAuthorizationRoleBinding(kubeclientset kubernetes.Interface, clientset csclientset.Interface, csInformerFactory csinformers.SharedInformerFactory, cloud cscloud.Interface) *AuthorizationRoleBindingSyncController {
	authorizationRoleBindingInformer := csInformerFactory.ContainershipAuth().V3().AuthorizationRoleBindings()

	authorizationRoleBindingInformer.Informer().AddIndexers(tools.IndexByIDKeyFun())

	return &AuthorizationRoleBindingSyncController{
		syncController: &syncController{
			name:      authorizationRoleBindingSyncControllerName,
			clientset: clientset,
			synced:    authorizationRoleBindingInformer.Informer().HasSynced,
			informer:  authorizationRoleBindingInformer.Informer(),
			recorder:  tools.CreateAndStartRecorder(kubeclientset, authorizationRoleBindingSyncControllerName),
		},

		lister:        authorizationRoleBindingInformer.Lister(),
		cloudResource: resources.NewCsAuthorizationRoleBindings(cloud),
	}
}

// SyncWithCloud kicks of the Sync() function, should be started only after
// Informer caches we are about to use are synced
func (c *AuthorizationRoleBindingSyncController) SyncWithCloud(stopCh <-chan struct{}) error {
	return c.syncWithCloud(c.doSync, stopCh)
}

func (c *AuthorizationRoleBindingSyncController) doSync() {
	log.Debug("Sync AuthorizationRoleBindings")
	// makes a request to containership api and write results to the resource's cache
	err := c.cloudResource.Sync()
	if err != nil {
		log.Error("AuthorizationRoleBindings failed to sync: ", err.Error())
		return
	}

	// write the cloud items by ID so we can easily see if anything needs
	// to be deleted
	cloudCacheByID := make(map[string]interface{}, 0)

	for _, cloudItem := range c.cloudResource.Cache() {
		cloudCacheByID[cloudItem.ID] = cloudItem

		// Try to find cloud item in CR cache
		// ByIndex() will only fail if the index name is invalid, i.e. a
		// programming error in our case.
		item, _ := c.informer.GetIndexer().ByIndex(tools.IndexByIDFunctionName, cloudItem.ID)
		if len(item) == 0 {
			log.Debugf("Cloud AuthorizationRoleBinding %s does not exist as CR - creating", cloudItem.ID)
			err = c.Create(cloudItem)
			if err != nil {
				log.Error("AuthorizationRoleBinding Create failed: ", err.Error())
			}
			continue
		}

		shouldUpdate := false
		authorizationRoleBindingCR := item[0].(*authv3.AuthorizationRoleBinding)
		// Any errors would be programming errors - just ignore them
		// TODO remove error return value from IsEqual() functions
		equal, _ := c.cloudResource.IsEqual(cloudItem, authorizationRoleBindingCR)
		if !equal {
			log.Debugf("Cloud AuthorizationRoleBinding %s does not match CR - will update", cloudItem.ID)
			shouldUpdate = true
		}

		// If a CR does not contain the required finalizer and is not marked
		// for deletion, it must be overwritten.
		if authorizationRoleBindingCR.DeletionTimestamp.IsZero() &&
			!tools.StringSliceContains(authorizationRoleBindingCR.Finalizers, constants.AuthorizationRoleBindingFinalizerName) {
			log.Debugf("Cloud AuthorizationRoleBinding %s is missing finalizer %s - will update",
				cloudItem.ID, constants.AuthorizationRoleBindingFinalizerName)
			shouldUpdate = true
		}

		if shouldUpdate {
			err = c.Update(cloudItem, authorizationRoleBindingCR)
			if err != nil {
				log.Error("AuthorizationRoleBinding Update failed: ", err.Error())
			}
		}
	}

	allCRs, err := c.lister.List(labels.NewSelector())
	if err != nil {
		log.Error(err)
		return
	}

	// Find CRs that do not exist in cloud
	for _, u := range allCRs {
		if _, exists := cloudCacheByID[u.Name]; !exists {
			log.Debugf("CR AuthorizationRoleBinding %s does not exist in cloud - deleting", u.Name)
			err = c.Delete(u.Namespace, u.Name)
			if err != nil {
				log.Error("AuthorizationRoleBinding Delete failed: ", err.Error())
			}
		}
	}
}

// Create takes a authorizationRoleBinding spec in cache and creates the CRD
func (c *AuthorizationRoleBindingSyncController) Create(l authv3.AuthorizationRoleBindingSpec) error {
	authorizationRoleBinding, err := c.clientset.ContainershipAuthV3().AuthorizationRoleBindings(constants.ContainershipNamespace).Create(&authv3.AuthorizationRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:       l.ID,
			Finalizers: []string{constants.AuthorizationRoleBindingFinalizerName},
		},
		Spec: l,
	})
	if err != nil {
		return err
	}

	// We can only fire an event if the object was successfully created,
	// otherwise there's no reasonable object to attach to.
	c.recorder.Event(authorizationRoleBinding, corev1.EventTypeNormal, "SyncCreate",
		"Detected missing CR")

	return nil
}

// Update takes a authorizationRoleBinding spec and updates the associated AuthorizationRoleBinding CR spec
// with the new values
func (c *AuthorizationRoleBindingSyncController) Update(l authv3.AuthorizationRoleBindingSpec, obj interface{}) error {
	authorizationRoleBinding, ok := obj.(*authv3.AuthorizationRoleBinding)
	if !ok {
		return fmt.Errorf("Error trying to use a non AuthorizationRoleBinding CR object to update a AuthorizationRoleBinding CR")
	}

	c.recorder.Event(authorizationRoleBinding, corev1.EventTypeNormal, "AuthorizationRoleBindingUpdate",
		"Detected change in Cloud, updating")

	pCopy := authorizationRoleBinding.DeepCopy()
	pCopy.Spec = l
	// Always add the required finalizer in case it was originally missing
	pCopy.Finalizers = tools.AddStringToSliceIfMissing(pCopy.Finalizers, constants.AuthorizationRoleBindingFinalizerName)

	_, err := c.clientset.ContainershipAuthV3().AuthorizationRoleBindings(constants.ContainershipNamespace).Update(pCopy)

	if err != nil {
		c.recorder.Eventf(authorizationRoleBinding, corev1.EventTypeWarning, "AuthorizationRoleBindingUpdateError",
			"Error updating: %s", err.Error())
	}

	return err
}

// Delete takes a name of the CR and deletes it
func (c *AuthorizationRoleBindingSyncController) Delete(namespace, name string) error {
	return c.clientset.ContainershipAuthV3().AuthorizationRoleBindings(namespace).Delete(name, &metav1.DeleteOptions{})
}
