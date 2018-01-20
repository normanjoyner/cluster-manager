package controller

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"

	containershipv3 "github.com/containership/cloud-agent/pkg/apis/containership.io/v3"
	csclientset "github.com/containership/cloud-agent/pkg/client/clientset/versioned"
	csinformers "github.com/containership/cloud-agent/pkg/client/informers/externalversions"
	cslisters "github.com/containership/cloud-agent/pkg/client/listers/containership.io/v3"

	"github.com/containership/cloud-agent/internal/constants"
	"github.com/containership/cloud-agent/internal/env"
	"github.com/containership/cloud-agent/internal/log"
	"github.com/containership/cloud-agent/internal/resources"
)

// PluginSyncController is the implementation for syncing Plugin CRDs
type PluginSyncController struct {
	// clientset is a clientset for our own API group
	clientset csclientset.Interface

	lister   cslisters.PluginLister
	synced   cache.InformerSynced
	informer cache.SharedIndexInformer

	cloudResource *resources.CsPlugins
}

// NewPlugin returns a PluginSyncController that will be in control of pulling from cloud
// comparing to the CRD cache and modifying based on those compares
func NewPlugin(csInformerFactory csinformers.SharedInformerFactory, clientset csclientset.Interface) *PluginSyncController {
	pluginInformer := csInformerFactory.Containership().V3().Plugins()

	pluginInformer.Informer().AddIndexers(indexByIDKeyFun())

	return &PluginSyncController{
		clientset:     clientset,
		lister:        pluginInformer.Lister(),
		synced:        pluginInformer.Informer().HasSynced,
		informer:      pluginInformer.Informer(),
		cloudResource: resources.NewCsPlugins(),
	}
}

// SyncWithCloud kicks of the Sync() function, should be started only after
// Informer caches we are about to use are synced
func (c *PluginSyncController) SyncWithCloud(stopCh <-chan struct{}) error {
	log.Info("Starting Plugin resource controller")

	log.Info("Waiting for Plugin informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.synced); !ok {
		return fmt.Errorf("Failed to wait for Plugin cache to sync")
	}

	// Only run one worker because a resource's underlying
	// cache is not thread-safe and we don't want to do parallel
	// requests to the API anyway
	go wait.JitterUntil(c.doSync,
		env.ContainershipCloudSyncInterval(),
		constants.SyncJitterFactor,
		true, // sliding: restart period only after doSync finishes
		stopCh)

	<-stopCh
	log.Info("Plugin sync stopped")

	return nil
}

func (c *PluginSyncController) doSync() {
	log.Debug("Sync Plugins")
	// makes a request to containership api and write results to the resource's cache
	err := resources.Sync(c.cloudResource)
	if err != nil {
		log.Error("Plugins failed to sync: ", err.Error())
	}

	// write the cloud items by ID so we can easily see if anything needs
	// to be deleted
	cloudCacheByID := make(map[string]interface{}, 0)

	for _, cloudItem := range c.cloudResource.Cache() {
		cloudCacheByID[cloudItem.ID] = cloudItem

		// Try to find cloud item in CR cache
		item, err := c.informer.GetIndexer().ByIndex("byID", cloudItem.ID)
		if err == nil && len(item) == 0 {
			log.Debugf("Cloud Plugin %s does not exist as CR - creating", cloudItem.ID)
			err = c.Create(cloudItem)
			if err != nil {
				log.Error("Plugin Create failed: ", err.Error())
			}
			continue
		}

		// TODO we would normally check for if the cloud object is different
		// from the cached object, but since we are not currently
		// supporting updating plugins we don't need to

	}

	allCRs, err := c.lister.List(labels.NewSelector())
	if err != nil {
		log.Error(err)
		return
	}

	// Find CRs that do not exist in cloud
	for _, u := range allCRs {
		if _, exists := cloudCacheByID[u.Name]; !exists {
			log.Debugf("CR Plugin %s does not exist in cloud - deleting", u.Name)
			err = c.Delete(u.Namespace, u.Name)
			if err != nil {
				log.Error("Plugin Delete failed: ", err.Error())
			}
		}
	}
}

// Create takes a plugin spec in cache and creates the CRD
func (c *PluginSyncController) Create(u containershipv3.PluginSpec) error {
	_, err := c.clientset.ContainershipV3().Plugins(constants.ContainershipNamespace).Create(&containershipv3.Plugin{
		ObjectMeta: metav1.ObjectMeta{
			Name: u.ID,
		},
		Spec: u,
	})

	return err
}

// Delete takes a name or the CRD and deletes it
func (c *PluginSyncController) Delete(namespace, name string) error {
	return c.clientset.ContainershipV3().Plugins(namespace).Delete(name, &metav1.DeleteOptions{})
}
