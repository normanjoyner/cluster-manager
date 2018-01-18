package controller

import (
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"

	containershipv3 "github.com/containership/cloud-agent/pkg/apis/containership.io/v3"
	csclientset "github.com/containership/cloud-agent/pkg/client/clientset/versioned"
	csinformers "github.com/containership/cloud-agent/pkg/client/informers/externalversions"
	cslisters "github.com/containership/cloud-agent/pkg/client/listers/containership.io/v3"

	"github.com/containership/cloud-agent/internal/constants"
	"github.com/containership/cloud-agent/internal/envvars"
	"github.com/containership/cloud-agent/internal/log"
	"github.com/containership/cloud-agent/internal/resources"
)

// UserController is the implementation for syncing User CRDs
type UserController struct {
	// clientset is a clientset for our own API group
	clientset csclientset.Interface

	lister   cslisters.UserLister
	synced   cache.InformerSynced
	informer cache.SharedIndexInformer

	cloudResource *resources.CsUsers
}

// NewUser returns a UserController that will be in control of pulling from cloud
// comparing to the CRD cache and modifying based on those compares
func NewUser(csInformerFactory csinformers.SharedInformerFactory, clientset csclientset.Interface) *UserController {
	userInformer := csInformerFactory.Containership().V3().Users()

	userInformer.Informer().AddIndexers(indexByIDKeyFun())

	return &UserController{
		clientset:     clientset,
		lister:        userInformer.Lister(),
		synced:        userInformer.Informer().HasSynced,
		informer:      userInformer.Informer(),
		cloudResource: resources.NewCsUsers(),
	}
}

// SyncWithCloud kicks of the Sync() function, should be started only after
// Informer caches we are about to use are synced
func (c *UserController) SyncWithCloud(stopCh <-chan struct{}) error {
	log.Info("Starting User resource controller")

	log.Info("Waiting for User informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.synced); !ok {
		return fmt.Errorf("Failed to wait for User cache to sync")
	}

	// Only run one worker because a resource's underlying
	// cache is not thread-safe and we don't want to do parallel
	// requests to the API anyway
	go wait.Until(c.doSync, time.Second*envvars.GetAgentSyncIntervalInSeconds(), stopCh)

	<-stopCh
	log.Info("User sync stopped")

	return nil
}

func (c *UserController) doSync() {
	log.Debug("Sync Users")
	// makes a request to containership api and write results to the resource's cache
	err := resources.Sync(c.cloudResource)
	if err != nil {
		log.Error("Users failed to sync: ", err.Error())
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
func (c *UserController) Create(u containershipv3.UserSpec) error {
	_, err := c.clientset.ContainershipV3().Users(constants.ContainershipNamespace).Create(&containershipv3.User{
		ObjectMeta: metav1.ObjectMeta{
			Name: u.ID,
		},
		Spec: u,
	})

	return err
}

// Delete takes a name or the CRD and deletes it
func (c *UserController) Delete(namespace, name string) error {
	return c.clientset.ContainershipV3().Users(namespace).Delete(name, &metav1.DeleteOptions{})
}

// Update takes a user spec in cache and updates a User CRD spec with the same
// ID with that value
func (c *UserController) Update(u containershipv3.UserSpec, obj interface{}) error {
	user, ok := obj.(*containershipv3.User)
	if !ok {
		return fmt.Errorf("Error trying to use a non User CRD object to update a User CRD")
	}

	uCopy := user.DeepCopy()
	uCopy.Spec = u

	_, err := c.clientset.ContainershipV3().Users(constants.ContainershipNamespace).Update(uCopy)

	return err
}
