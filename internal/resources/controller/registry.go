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

	cloudResource         *resources.CsRegistries
	tokenRegenerationByID map[string]chan bool
}

// NewRegistry returns a RegistryController that will be in control of pulling from cloud
// comparing to the CRD cache and modifying based on those compares
func NewRegistry(csInformerFactory csinformers.SharedInformerFactory, clientset csclientset.Interface) *RegistryController {
	registryInformer := csInformerFactory.Containership().V3().Registries()

	registryInformer.Informer().AddIndexers(indexByIDKeyFun())

	return &RegistryController{
		clientset:             clientset,
		lister:                registryInformer.Lister(),
		synced:                registryInformer.Informer().HasSynced,
		informer:              registryInformer.Informer(),
		cloudResource:         resources.NewCsRegistries(),
		tokenRegenerationByID: make(map[string]chan bool, 0),
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
	token, err := c.cloudResource.GetAuthToken(u)
	if err != nil {
		return err
	}

	u.AuthToken = token
	newReg, err := c.clientset.ContainershipV3().Registries(constants.ContainershipNamespace).Create(&containershipv3.Registry{
		ObjectMeta: metav1.ObjectMeta{
			Name: u.ID,
		},
		Spec: u,
	})

	if err != nil {
		return err
	}

	if newReg.Spec.Provider == constants.EC2Registry {
		c.tokenRegenerationByID[newReg.Name] = c.watchToken(newReg)
	}

	return nil
}

// Delete takes a name or the CRD and deletes it
func (c *RegistryController) Delete(namespace, name string) error {
	err := c.clientset.ContainershipV3().Registries(namespace).Delete(name, &metav1.DeleteOptions{})

	if err != nil {
		return err
	}

	// If there was not an issue deleting the registry, if there is a routine to
	// sync auth token, stop it
	if t, ok := c.tokenRegenerationByID[name]; ok {
		safeClose(t)
	}

	return nil
}

// Update takes a registry spec in cache and updates a Registry CRD spec with the same
// ID with that value
func (c *RegistryController) Update(r containershipv3.RegistrySpec, obj interface{}) error {
	registry, ok := obj.(*containershipv3.Registry)
	if !ok {
		return fmt.Errorf("Error trying to use a non Registry CRD object to update a Registry CRD")
	}

	if t, ok := c.tokenRegenerationByID[r.ID]; ok {
		safeClose(t)
	}

	token, err := c.cloudResource.GetAuthToken(r)
	if err != nil {
		return err
	}

	r.AuthToken = token
	rCopy := registry.DeepCopy()
	rCopy.Spec = r
	newReg, err := c.clientset.ContainershipV3().Registries(constants.ContainershipNamespace).Update(rCopy)

	if err != nil {
		return err
	}

	if newReg.Spec.Provider == constants.EC2Registry {
		c.tokenRegenerationByID[r.ID] = c.watchToken(newReg)
	}

	return nil
}

// Takes a channel of type bool, if it's open it closes it, otherwise it ignores
// the channel. Trying to close a channel that is already closed results in a panic
func safeClose(t chan bool) {
	select {
	case _, ok := <-t:
		if ok {
			close(t)
		} else {
			fmt.Println("Channel closed!")
		}
	default:
		close(t)
	}
}

// watchToken takes a registry, waits 11 hours, make a request to get the registry
// then passes it to update so it can get a new AuthToken assigned to it
func (c *RegistryController) watchToken(r *containershipv3.Registry) chan bool {
	stop := make(chan bool)

	go func() {
		t := time.NewTicker(time.Second * 11)
		for {
			select {
			case <-t.C:
				rObj, err := c.lister.Registries(r.Namespace).Get(r.Name)
				if err != nil {
					log.Error("Could not get the registry, to update auth token: ", err.Error())
					safeClose(stop)
					return
				}

				err = c.Update(rObj.Spec, rObj)
				if err != nil {
					log.Error("Could not update the registry auth token: ", err.Error())
					safeClose(stop)
					return
				}

				safeClose(stop)
			case <-stop:
				t.Stop()
				safeClose(stop)
				return
			}
		}
	}()

	return stop
}
