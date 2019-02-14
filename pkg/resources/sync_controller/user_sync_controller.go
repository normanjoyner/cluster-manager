package synccontroller

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"

	cscloud "github.com/containership/csctl/cloud"

	csv3 "github.com/containership/cluster-manager/pkg/apis/containership.io/v3"
	csclientset "github.com/containership/cluster-manager/pkg/client/clientset/versioned"
	csinformers "github.com/containership/cluster-manager/pkg/client/informers/externalversions"
	cslisters "github.com/containership/cluster-manager/pkg/client/listers/containership.io/v3"

	"github.com/containership/cluster-manager/pkg/constants"
	"github.com/containership/cluster-manager/pkg/log"
	"github.com/containership/cluster-manager/pkg/resources"
	"github.com/containership/cluster-manager/pkg/tools"
)

// UserSyncController is the implementation for syncing User CRDs
type UserSyncController struct {
	*syncController

	lister        cslisters.UserLister
	cloudResource *resources.CsUsers
}

const (
	userSyncControllerName = "UserSyncController"
)

// NewUser returns a UserSyncController that will be in control of pulling from cloud
// comparing to the CRD cache and modifying based on those compares
func NewUser(kubeclientset kubernetes.Interface, clientset csclientset.Interface, csInformerFactory csinformers.SharedInformerFactory, cloud cscloud.Interface) *UserSyncController {
	userInformer := csInformerFactory.Containership().V3().Users()

	userInformer.Informer().AddIndexers(tools.IndexByIDKeyFun())

	return &UserSyncController{
		syncController: &syncController{
			name:      userSyncControllerName,
			clientset: clientset,
			synced:    userInformer.Informer().HasSynced,
			informer:  userInformer.Informer(),
			recorder:  tools.CreateAndStartRecorder(kubeclientset, userSyncControllerName),
		},

		lister:        userInformer.Lister(),
		cloudResource: resources.NewCsUsers(cloud),
	}
}

// SyncWithCloud kicks of the Sync() function, should be started only after
// Informer caches we are about to use are synced
func (c *UserSyncController) SyncWithCloud(stopCh <-chan struct{}) error {
	return c.syncWithCloud(c.doSync, stopCh)
}

func (c *UserSyncController) doSync() {
	log.Debug("Sync Users")
	// makes a request to containership api and write results to the resource's cache
	err := c.cloudResource.Sync()
	if err != nil {
		log.Error("Users failed to sync: ", err.Error())
		return
	}

	// write the cloud items by ID so we can easily see if anything needs
	// to be deleted
	cloudCacheByID := make(map[string]interface{}, 0)

	for _, cloudItem := range c.cloudResource.Cache() {
		cloudCacheByID[cloudItem.ID] = cloudItem

		// Try to find cloud item in CR cache
		item, err := c.informer.GetIndexer().ByIndex("byID", cloudItem.ID)
		if err == nil && len(item) == 0 {
			log.Debugf("Cloud User %s does not exist as CR - creating", cloudItem.ID)
			err = c.Create(cloudItem)
			if err != nil {
				log.Error("User Create failed: ", err.Error())
			}
			continue
		}

		// We only need to pass in the first index of item since the key by function
		// is keying by a unique value
		// Only update if err == nil because if err != nil then the types are
		// incorrect somehow and we shouldn't update.
		if equal, err := c.cloudResource.IsEqual(cloudItem, item[0]); err == nil && !equal {
			log.Debugf("Cloud User %s does not match CR - updating", cloudItem.ID)
			err = c.Update(cloudItem, item[0])
			if err != nil {
				log.Error("User Update failed: ", err.Error())
			}
			continue
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
			log.Debugf("CR User %s does not exist in cloud - deleting", u.Name)
			err = c.Delete(u.Namespace, u.Name)
			if err != nil {
				log.Error("User Delete failed: ", err.Error())
			}
		}
	}
}

// Create takes a user spec in cache and creates the CRD
func (c *UserSyncController) Create(u csv3.UserSpec) error {
	user, err := c.clientset.ContainershipV3().Users(constants.ContainershipNamespace).Create(&csv3.User{
		ObjectMeta: metav1.ObjectMeta{
			Name: u.ID,
		},
		Spec: u,
	})
	if err != nil {
		return err
	}

	// We can only fire an event if the object was successfully created,
	// otherwise there's no reasonable object to attach to.
	c.recorder.Event(user, corev1.EventTypeNormal, "SyncCreate",
		"Detected missing CR")

	return nil
}

// Delete takes a name or the CRD and deletes it
func (c *UserSyncController) Delete(namespace, name string) error {
	return c.clientset.ContainershipV3().Users(namespace).Delete(name, &metav1.DeleteOptions{})
}

// Update takes a user spec in cache and updates a User CRD spec with the same
// ID with that value
func (c *UserSyncController) Update(u csv3.UserSpec, obj interface{}) error {
	user, ok := obj.(*csv3.User)
	if !ok {
		return fmt.Errorf("Error trying to use a non User CRD object to update a User CRD")
	}

	c.recorder.Event(user, corev1.EventTypeNormal, "SyncUpdate",
		"Detected change in Cloud, updating")

	uCopy := user.DeepCopy()
	uCopy.Spec = u

	_, err := c.clientset.ContainershipV3().Users(constants.ContainershipNamespace).Update(uCopy)

	if err != nil {
		c.recorder.Eventf(user, corev1.EventTypeNormal, "SyncUpdateError",
			"Error updating: %s", err.Error())
	}

	return err
}
