package controller

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
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

// New returns a UserController that will be in control of pulling from cloud
// comparing to the CRD cach and modifying based on those compares
func New(csInformerFactory csinformers.SharedInformerFactory, clientset csclientset.Interface) *UserController {
	userInformer := csInformerFactory.Containership().V3().Users()

	userInformer.Informer().AddIndexers(cache.Indexers{
		"byID": func(obj interface{}) ([]string, error) {
			meta, err := meta.Accessor(obj)
			if err != nil {
				return []string{""}, fmt.Errorf("object has no meta: %v", err)
			}
			return []string{meta.GetName()}, nil
		},
	})

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

	log.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.synced); !ok {
		return fmt.Errorf("Failed to wait for caches to sync")
	}

	log.Info("Starting workers")
	numWorkers := 1
	// Launch two workers to process User resources
	for i := 0; i < numWorkers; i++ {
		go wait.Until(c.doSync, time.Second*envvars.GetAgentSyncIntervalInSeconds(), stopCh)
	}

	log.Info("Started workers")
	<-stopCh
	log.Info("Shutting down workers")

	return nil
}

func (c *UserController) doSync() {
	log.Info("Starting usercontroller doSync()")
	// makes a request to containership api and write results to the resouce's cache
	err := resources.Sync(c.cloudResource)

	if err != nil {
		log.Error("Failed to sync")
	}

	// write the cloud items by ID so we can easily see if anything needs
	// to be deleted
	cloudCacheByID := make(map[string]interface{}, 0)

	for _, cloudItem := range c.cloudResource.Cache() {
		cloudCacheByID[cloudItem.ID] = cloudItem

		item, err := c.informer.GetIndexer().ByIndex("byID", cloudItem.ID)
		if err == nil && len(item) == 0 {
			err = c.Create(cloudItem)
			if err != nil {
				log.Error(err)
			}
			continue
		}

		if equal, err := c.cloudResource.IsEqual(cloudItem, item); err != nil && !equal {
			err = c.Update(cloudItem, item)
			if err != nil {
				log.Error(err)
			}
			continue
		}
	}

	allUserCRDS, err := c.lister.List(labels.NewSelector())
	if err != nil {
		log.Error(err)
		return
	}

	for _, u := range allUserCRDS {
		if _, exists := cloudCacheByID[u.Name]; !exists {
			err = c.Delete(u.Name)
			if err != nil {
				log.Error(err)
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
func (c *UserController) Delete(name string) error {
	return c.clientset.ContainershipV3().Users(constants.ContainershipNamespace).Delete(name, &metav1.DeleteOptions{})
}

// Update takes a user spec in cache and updates a User CRD spec with the same
// ID with that value
func (c *UserController) Update(u containershipv3.UserSpec, obj interface{}) error {
	user, ok := obj.(*containershipv3.User)
	if !ok {
		return fmt.Errorf("Error tying to use a non User CRD object to update a User CRD")
	}

	uCopy := user.DeepCopy()
	uCopy.Spec = u

	_, err := c.clientset.ContainershipV3().Users(constants.ContainershipNamespace).Update(uCopy)

	return err
}
