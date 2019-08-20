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

// AuthorizationRoleSyncController is the implementation for syncing AuthorizationRole CRDs
type AuthorizationRoleSyncController struct {
	*syncController

	lister        authlisters.AuthorizationRoleLister
	cloudResource *resources.CsAuthorizationRoles
}

const (
	authorizationRoleSyncControllerName = "AuthorizationRoleSyncController"
)

// NewAuthorizationRole returns a AuthorizationRoleSyncController that will be in control of pulling from cloud
// comparing to the CRD cache and modifying based on those compares
func NewAuthorizationRole(kubeclientset kubernetes.Interface, clientset csclientset.Interface, csInformerFactory csinformers.SharedInformerFactory, cloud cscloud.Interface) *AuthorizationRoleSyncController {
	authorizationRoleInformer := csInformerFactory.ContainershipAuth().V3().AuthorizationRoles()

	authorizationRoleInformer.Informer().AddIndexers(tools.IndexByIDKeyFun())

	return &AuthorizationRoleSyncController{
		syncController: &syncController{
			name:      authorizationRoleSyncControllerName,
			clientset: clientset,
			synced:    authorizationRoleInformer.Informer().HasSynced,
			informer:  authorizationRoleInformer.Informer(),
			recorder:  tools.CreateAndStartRecorder(kubeclientset, authorizationRoleSyncControllerName),
		},

		lister:        authorizationRoleInformer.Lister(),
		cloudResource: resources.NewCsAuthorizationRoles(cloud),
	}
}

// SyncWithCloud kicks of the Sync() function, should be started only after
// Informer caches we are about to use are synced
func (c *AuthorizationRoleSyncController) SyncWithCloud(stopCh <-chan struct{}) error {
	return c.syncWithCloud(c.doSync, stopCh)
}

func (c *AuthorizationRoleSyncController) doSync() {
	log.Debug("Sync AuthorizationRoles")
	// makes a request to containership api and write results to the resource's cache
	err := c.cloudResource.Sync()
	if err != nil {
		log.Error("AuthorizationRoles failed to sync: ", err.Error())
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
			log.Debugf("Cloud AuthorizationRole %s does not exist as CR - creating", cloudItem.ID)
			err = c.Create(cloudItem)
			if err != nil {
				log.Error("AuthorizationRole Create failed: ", err.Error())
			}
			continue
		}

		shouldUpdate := false
		authorizationRoleCR := item[0].(*authv3.AuthorizationRole)
		// Any errors would be programming errors - just ignore them
		// TODO remove error return value from IsEqual() functions
		equal, _ := c.cloudResource.IsEqual(cloudItem, authorizationRoleCR)
		if !equal {
			log.Debugf("Cloud AuthorizationRole %s does not match CR - will update", cloudItem.ID)
			shouldUpdate = true
		}

		// If a CR does not contain the required finalizer and is not marked
		// for deletion, it must be overwritten.
		if authorizationRoleCR.DeletionTimestamp.IsZero() &&
			!tools.StringSliceContains(authorizationRoleCR.Finalizers, constants.AuthorizationRoleFinalizerName) {
			log.Debugf("Cloud AuthorizationRole %s is missing finalizer %s - will update",
				cloudItem.ID, constants.AuthorizationRoleFinalizerName)
			shouldUpdate = true
		}

		if shouldUpdate {
			err = c.Update(cloudItem, authorizationRoleCR)
			if err != nil {
				log.Error("AuthorizationRole Update failed: ", err.Error())
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
			log.Debugf("CR AuthorizationRole %s does not exist in cloud - deleting", u.Name)
			err = c.Delete(u.Namespace, u.Name)
			if err != nil {
				log.Error("AuthorizationRole Delete failed: ", err.Error())
			}
		}
	}
}

// Create takes a authorizationRole spec in cache and creates the CRD
func (c *AuthorizationRoleSyncController) Create(l authv3.AuthorizationRoleSpec) error {
	authorizationRole, err := c.clientset.ContainershipAuthV3().AuthorizationRoles(constants.ContainershipNamespace).Create(&authv3.AuthorizationRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:       l.ID,
			Finalizers: []string{constants.AuthorizationRoleFinalizerName},
		},
		Spec: l,
	})
	if err != nil {
		return err
	}

	// We can only fire an event if the object was successfully created,
	// otherwise there's no reasonable object to attach to.
	c.recorder.Event(authorizationRole, corev1.EventTypeNormal, "SyncCreate",
		"Detected missing CR")

	return nil
}

// Update takes a authorizationRole spec and updates the associated AuthorizationRole CR spec
// with the new values
func (c *AuthorizationRoleSyncController) Update(l authv3.AuthorizationRoleSpec, obj interface{}) error {
	authorizationRole, ok := obj.(*authv3.AuthorizationRole)
	if !ok {
		return fmt.Errorf("Error trying to use a non AuthorizationRole CR object to update a AuthorizationRole CR")
	}

	c.recorder.Event(authorizationRole, corev1.EventTypeNormal, "AuthorizationRoleUpdate",
		"Detected change in Cloud, updating")

	pCopy := authorizationRole.DeepCopy()
	pCopy.Spec = l
	// Always add the required finalizer in case it was originally missing
	pCopy.Finalizers = tools.AddStringToSliceIfMissing(pCopy.Finalizers, constants.AuthorizationRoleFinalizerName)

	_, err := c.clientset.ContainershipAuthV3().AuthorizationRoles(constants.ContainershipNamespace).Update(pCopy)

	if err != nil {
		c.recorder.Eventf(authorizationRole, corev1.EventTypeWarning, "AuthorizationRoleUpdateError",
			"Error updating: %s", err.Error())
	}

	return err
}

// Delete takes a name of the CR and deletes it
func (c *AuthorizationRoleSyncController) Delete(namespace, name string) error {
	return c.clientset.ContainershipAuthV3().AuthorizationRoles(namespace).Delete(name, &metav1.DeleteOptions{})
}
