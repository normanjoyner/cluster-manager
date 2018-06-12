package synccontroller

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"

	containershipv3 "github.com/containership/cloud-agent/pkg/apis/containership.io/v3"
	csclientset "github.com/containership/cloud-agent/pkg/client/clientset/versioned"
	csinformers "github.com/containership/cloud-agent/pkg/client/informers/externalversions"
	cslisters "github.com/containership/cloud-agent/pkg/client/listers/containership.io/v3"

	"github.com/containership/cloud-agent/pkg/constants"
	"github.com/containership/cloud-agent/pkg/log"
	"github.com/containership/cloud-agent/pkg/resources"
	"github.com/containership/cloud-agent/pkg/tools"
)

// PluginSyncController is the implementation for syncing Plugin CRDs
type PluginSyncController struct {
	*syncController

	lister        cslisters.PluginLister
	cloudResource *resources.CsPlugins
}

const (
	pluginSyncControllerName = "PluginSyncController"
)

// NewPlugin returns a PluginSyncController that will be in control of pulling from cloud
// comparing to the CRD cache and modifying based on those compares
func NewPlugin(kubeclientset kubernetes.Interface, clientset csclientset.Interface, csInformerFactory csinformers.SharedInformerFactory) *PluginSyncController {
	pluginInformer := csInformerFactory.Containership().V3().Plugins()

	pluginInformer.Informer().AddIndexers(tools.IndexByIDKeyFun())

	return &PluginSyncController{
		syncController: &syncController{
			name:      pluginSyncControllerName,
			clientset: clientset,
			synced:    pluginInformer.Informer().HasSynced,
			informer:  pluginInformer.Informer(),
			recorder:  tools.CreateAndStartRecorder(kubeclientset, pluginSyncControllerName),
		},

		lister:        pluginInformer.Lister(),
		cloudResource: resources.NewCsPlugins(),
	}
}

// SyncWithCloud kicks of the Sync() function, should be started only after
// Informer caches we are about to use are synced
func (c *PluginSyncController) SyncWithCloud(stopCh <-chan struct{}) error {
	return c.syncWithCloud(c.doSync, stopCh)
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
func (c *PluginSyncController) Create(p containershipv3.PluginSpec) error {
	plugin, err := c.clientset.ContainershipV3().Plugins(constants.ContainershipNamespace).Create(&containershipv3.Plugin{
		ObjectMeta: metav1.ObjectMeta{
			Name: p.ID,
		},
		Spec: p,
	})
	if err != nil {
		return err
	}

	// We can only fire an event if the object was successfully created,
	// otherwise there's no reasonable object to attach to.
	c.recorder.Event(plugin, corev1.EventTypeNormal, "SyncCreate",
		"Detected missing CR")

	return nil
}

// Delete takes a name or the CRD and deletes it
func (c *PluginSyncController) Delete(namespace, name string) error {
	return c.clientset.ContainershipV3().Plugins(namespace).Delete(name, &metav1.DeleteOptions{})
}
