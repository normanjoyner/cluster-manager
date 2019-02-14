package synccontroller

import (
	"encoding/json"
	"fmt"

	cscloud "github.com/containership/csctl/cloud"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"

	csv3 "github.com/containership/cluster-manager/pkg/apis/containership.io/v3"
	csclientset "github.com/containership/cluster-manager/pkg/client/clientset/versioned"
	csinformers "github.com/containership/cluster-manager/pkg/client/informers/externalversions"
	cslisters "github.com/containership/cluster-manager/pkg/client/listers/containership.io/v3"

	"github.com/containership/cluster-manager/pkg/constants"
	"github.com/containership/cluster-manager/pkg/log"
	"github.com/containership/cluster-manager/pkg/resources"
	"github.com/containership/cluster-manager/pkg/tools"
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
func NewPlugin(kubeclientset kubernetes.Interface, clientset csclientset.Interface, csInformerFactory csinformers.SharedInformerFactory, cloud cscloud.Interface) *PluginSyncController {
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
		cloudResource: resources.NewCsPlugins(cloud),
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
	err := c.cloudResource.Sync()
	if err != nil {
		log.Error("Plugins failed to sync: ", err.Error())
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
			log.Debugf("Cloud Plugin %s does not exist as CR - creating", cloudItem.ID)
			err = c.Create(cloudItem)
			if err != nil {
				log.Error("Plugin Create failed: ", err.Error())
			}
			continue
		}

		pluginCR := item[0]
		if equal, err := c.cloudResource.IsEqual(cloudItem, pluginCR); err == nil && !equal {
			log.Debugf("Cloud Plugin %s does not match CR - updating", cloudItem.ID)
			err = c.Update(cloudItem, pluginCR)
			if err != nil {
				log.Error("Plugin Update failed: ", err.Error())
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
			log.Debugf("CR Plugin %s does not exist in cloud - deleting", u.Name)
			err = c.Delete(u.Namespace, u.Name)
			if err != nil {
				log.Error("Plugin Delete failed: ", err.Error())
			}
		}
	}
}

// Create takes a plugin spec in cache and creates the CRD
func (c *PluginSyncController) Create(p csv3.PluginSpec) error {
	plugin, err := c.clientset.ContainershipV3().Plugins(constants.ContainershipNamespace).Create(&csv3.Plugin{
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

// Update takes a plugin spec and updates the associated Plugin CR spec
// with the new values
func (c *PluginSyncController) Update(p csv3.PluginSpec, obj interface{}) error {
	plugin, ok := obj.(*csv3.Plugin)
	if !ok {
		return fmt.Errorf("Error trying to use a non Plugin CR object to update a Plugin CR")
	}

	c.recorder.Event(plugin, corev1.EventTypeNormal, "PluginUpdate",
		"Detected change in Cloud, updating")

	pCopy := plugin.DeepCopy()
	pCopy.Spec = p
	setPluginHistoryAnnotation(plugin, pCopy)

	_, err := c.clientset.ContainershipV3().Plugins(constants.ContainershipNamespace).Update(pCopy)

	if err != nil {
		c.recorder.Eventf(plugin, corev1.EventTypeWarning, "PluginUpdateError",
			"Error updating: %s", err.Error())
	}

	return err
}

func setPluginHistoryAnnotation(plugin *csv3.Plugin, pCopy *csv3.Plugin) {
	history := make([]csv3.PluginSpec, 0)
	ann, ok := plugin.Annotations[constants.PluginHistoryAnnotation]
	var err error
	if ok && ann != "" {
		err = json.Unmarshal([]byte(ann), &history)
	}

	// If there is an error unmarshalling the history we want to clear it and
	// start fresh with the new history that is formatted correctly
	if err != nil {
		history = make([]csv3.PluginSpec, 0)
	}

	history = append(history, plugin.Spec)
	sHistory, _ := json.Marshal(history)

	if pCopy.Annotations == nil {
		pCopy.Annotations = make(map[string]string, 0)
	}

	pCopy.Annotations[constants.PluginHistoryAnnotation] = string(sHistory)
}

// Delete takes a name or the CRD and deletes it
func (c *PluginSyncController) Delete(namespace, name string) error {
	return c.clientset.ContainershipV3().Plugins(namespace).Delete(name, &metav1.DeleteOptions{})
}
