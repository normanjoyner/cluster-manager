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

// RegistryController is the implementation for syncing Registry CRDs
type RegistryController struct {
	// clientset is a clientset for our own API group
	clientset csclientset.Interface

	lister   cslisters.RegistryLister
	synced   cache.InformerSynced
	informer cache.SharedIndexInformer

	cloudResource *resources.CsRegistries
}

// NewRegistry returns a RegistryController that will be in control of pulling from cloud
// comparing to the CRD cache and modifying based on those compares
func NewRegistry(csInformerFactory csinformers.SharedInformerFactory, clientset csclientset.Interface) *RegistryController {
	registryInformer := csInformerFactory.Containership().V3().Registries()

	registryInformer.Informer().AddIndexers(indexByIDKeyFun())

	return &RegistryController{
		clientset:     clientset,
		lister:        registryInformer.Lister(),
		synced:        registryInformer.Informer().HasSynced,
		informer:      registryInformer.Informer(),
		cloudResource: resources.NewCsRegistries(),
	}
}

// SyncWithCloud kicks of the Sync() function, should be started only after
// Informer caches we are about to use are synced
func (c *RegistryController) SyncWithCloud(stopCh <-chan struct{}) error {
	log.Info("Starting Registry resource controller")

	log.Info("Waiting for Registry informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.synced); !ok {
		return fmt.Errorf("Failed to wait for Registry cache to sync")
	}

	// Only run one worker because a resource's underlying
	// cache is not thread-safe and we don't want to do parallel
	// requests to the API anyway
	go wait.Until(c.doSync, time.Second*envvars.GetAgentSyncIntervalInSeconds(), stopCh)

	<-stopCh
	log.Info("Registry sync stopped")

	return nil
}

func (c *RegistryController) doSync() {
	// makes a request to containership api and write results to the resource's cache
	err := resources.Sync(c.cloudResource)

	if err != nil {
		log.Error("Registries failed to sync:", err.Error())
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
				log.Error("Registry Create failed:", err.Error())
			}
			continue
		}

		// We only need to pass in the first index of item since the key by function
		// is keying by a unique value
		if equal, err := c.cloudResource.IsEqual(cloudItem, item[0]); err != nil && !equal {
			err = c.Update(cloudItem, item[0])
			if err != nil {
				log.Error("Registry Update failed:", err.Error())
			}
			continue
		}
	}

	allRegistryCRDS, err := c.lister.List(labels.NewSelector())
	if err != nil {
		log.Error(err)
		return
	}

	for _, u := range allRegistryCRDS {
		if _, exists := cloudCacheByID[u.Name]; !exists {
			err = c.Delete(u.Namespace, u.Name)
			if err != nil {
				log.Error("Registry Delete failed:", err.Error())
			}
		}
	}

}

// Create takes a registry spec in cache and creates the CRD
func (c *RegistryController) Create(u containershipv3.RegistrySpec) error {
	// TODO :// add job for regreshing
	token, err := c.cloudResource.GetAuthToken(u)
	if err != nil {
		return err
	}

	u.AuthToken = token
	_, err = c.clientset.ContainershipV3().Registries(constants.ContainershipNamespace).Create(&containershipv3.Registry{
		ObjectMeta: metav1.ObjectMeta{
			Name: u.ID,
		},
		Spec: u,
	})

	return err
}

// Delete takes a name or the CRD and deletes it
func (c *RegistryController) Delete(namespace, name string) error {
	return c.clientset.ContainershipV3().Registries(namespace).Delete(name, &metav1.DeleteOptions{})
}

// Update takes a registry spec in cache and updates a Registry CRD spec with the same
// ID with that value
func (c *RegistryController) Update(r containershipv3.RegistrySpec, obj interface{}) error {
	registry, ok := obj.(*containershipv3.Registry)
	if !ok {
		return fmt.Errorf("Error trying to use a non Registry CRD object to update a Registry CRD")
	}

	token, err := c.cloudResource.GetAuthToken(r)
	if err != nil {
		return err
	}

	r.AuthToken = token
	rCopy := registry.DeepCopy()
	rCopy.Spec = r

	_, err = c.clientset.ContainershipV3().Registries(constants.ContainershipNamespace).Update(rCopy)

	return err
}
