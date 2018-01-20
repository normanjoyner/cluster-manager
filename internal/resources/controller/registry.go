package controller

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	containershipv3 "github.com/containership/cloud-agent/pkg/apis/containership.io/v3"
	csclientset "github.com/containership/cloud-agent/pkg/client/clientset/versioned"
	csscheme "github.com/containership/cloud-agent/pkg/client/clientset/versioned/scheme"
	csinformers "github.com/containership/cloud-agent/pkg/client/informers/externalversions"
	cslisters "github.com/containership/cloud-agent/pkg/client/listers/containership.io/v3"

	"github.com/containership/cloud-agent/internal/constants"
	"github.com/containership/cloud-agent/internal/envvars"
	"github.com/containership/cloud-agent/internal/log"
	"github.com/containership/cloud-agent/internal/resources"
)

// RegistrySyncController is the implementation for syncing Registry CRDs
type RegistrySyncController struct {
	// clientset is a clientset for our own API group
	clientset csclientset.Interface

	lister   cslisters.RegistryLister
	synced   cache.InformerSynced
	informer cache.SharedIndexInformer

	cloudResource         *resources.CsRegistries
	tokenRegenerationByID map[string]chan bool

	recorder record.EventRecorder
}

const (
	registrySyncControllerName = "RegistrySyncController"
)

// NewRegistry returns a RegistrySyncController that will be in control of pulling from cloud
// comparing to the CRD cache and modifying based on those compares
func NewRegistry(csInformerFactory csinformers.SharedInformerFactory, clientset csclientset.Interface) *RegistrySyncController {
	registryInformer := csInformerFactory.Containership().V3().Registries()

	registryInformer.Informer().AddIndexers(indexByIDKeyFun())

	// TODO we should not need to add to scheme everywhere. Pick a place.
	csscheme.AddToScheme(scheme.Scheme)

	log.Info(registrySyncControllerName, ": Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(log.Infof)
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: registrySyncControllerName})

	return &RegistrySyncController{
		clientset:             clientset,
		lister:                registryInformer.Lister(),
		synced:                registryInformer.Informer().HasSynced,
		informer:              registryInformer.Informer(),
		cloudResource:         resources.NewCsRegistries(),
		tokenRegenerationByID: make(map[string]chan bool, 0),
		recorder:              recorder,
	}
}

// SyncWithCloud kicks of the Sync() function, should be started only after
// Informer caches we are about to use are synced
func (c *RegistrySyncController) SyncWithCloud(stopCh <-chan struct{}) error {
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

func (c *RegistrySyncController) doSync() {
	log.Debug("Sync Registries")
	// makes a request to containership api and write results to the resource's cache
	err := resources.Sync(c.cloudResource)
	if err != nil {
		log.Error("Registries failed to sync: ", err.Error())
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
			log.Debugf("Cloud Registry %s does not exist as CR - creating", cloudItem.ID)
			err = c.Create(cloudItem)
			if err != nil {
				log.Error("Registry Create failed: ", err.Error())
			}
			continue
		}

		// We only need to pass in the first index of item since the key by function
		// is keying by a unique value
		// Only update if err == nil because if err != nil then the types are
		// incorrect somehow and we shouldn't update.
		if equal, err := c.cloudResource.IsEqual(cloudItem, item[0]); err == nil && !equal {
			log.Debugf("Cloud Registry %s does not match CR - updating", cloudItem.ID)
			log.Debugf("Cloud: %+v, Cache: %+v", cloudItem, item[0])
			err = c.Update(cloudItem, item[0])
			if err != nil {
				log.Error("Registry Update failed: ", err.Error())
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
			log.Debugf("CR Registry %s does not exist in cloud - deleting", u.Name)
			err = c.Delete(u.Namespace, u.Name)
			if err != nil {
				log.Error("Registry Delete failed: ", err.Error())
			}
		}
	}

}

// Create takes a registry spec in cache and creates the CRD
func (c *RegistrySyncController) Create(u containershipv3.RegistrySpec) error {
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
func (c *RegistrySyncController) Delete(namespace, name string) error {
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
func (c *RegistrySyncController) Update(r containershipv3.RegistrySpec, obj interface{}) error {
	registry, ok := obj.(*containershipv3.Registry)
	if !ok {
		return fmt.Errorf("Error trying to use a non Registry CRD object to update a Registry CRD")
	}

	c.recorder.Event(registry, corev1.EventTypeNormal, "SyncUpdate",
		"Detected change in Cloud, updating")

	rCopy := registry.DeepCopy()
	rCopy.Spec = r
	_, err := c.clientset.ContainershipV3().Registries(constants.ContainershipNamespace).Update(rCopy)
	if err != nil {
		c.recorder.Eventf(registry, corev1.EventTypeNormal, "SyncUpdateError",
			"Error updating: %s", err.Error())
	}

	return err
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

// watchToken takes a registry and waits, before the token on a registry becomes
// invalid it deletes the registry. Once deleted it will be recreated on the
// next sync with a new AuthToken
func (c *RegistrySyncController) watchToken(r *containershipv3.Registry) chan bool {
	stop := make(chan bool)

	go func() {
		t := time.NewTicker(time.Hour * 11)
		for {
			select {
			case <-t.C:
				c.recorder.Event(r, corev1.EventTypeNormal, "RegenerateAuthToken",
					"Timer expired, deleting registry so it will be regenerated")

				err := c.clientset.ContainershipV3().Registries(r.Namespace).
					Delete(r.Name, &metav1.DeleteOptions{})

				if err != nil {
					c.recorder.Eventf(r, corev1.EventTypeWarning, "RegenerateAuthTokenError",
						"Error deleting registry: %s", err.Error())
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
