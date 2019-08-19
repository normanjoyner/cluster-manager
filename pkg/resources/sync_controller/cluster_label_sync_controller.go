package synccontroller

import (
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

// ClusterLabelSyncController is the implementation for syncing ClusterLabel CRDs
type ClusterLabelSyncController struct {
	*syncController

	lister        cslisters.ClusterLabelLister
	cloudResource *resources.CsClusterLabels
}

const (
	clusterLabelSyncControllerName = "ClusterLabelSyncController"
)

// NewClusterLabel returns a ClusterLabelSyncController that will be in control of pulling from cloud
// comparing to the CRD cache and modifying based on those compares
func NewClusterLabel(kubeclientset kubernetes.Interface, clientset csclientset.Interface, csInformerFactory csinformers.SharedInformerFactory, cloud cscloud.Interface) *ClusterLabelSyncController {
	clusterLabelInformer := csInformerFactory.Containership().V3().ClusterLabels()

	clusterLabelInformer.Informer().AddIndexers(tools.IndexByIDKeyFun())

	return &ClusterLabelSyncController{
		syncController: &syncController{
			name:      clusterLabelSyncControllerName,
			clientset: clientset,
			synced:    clusterLabelInformer.Informer().HasSynced,
			informer:  clusterLabelInformer.Informer(),
			recorder:  tools.CreateAndStartRecorder(kubeclientset, clusterLabelSyncControllerName),
		},

		lister:        clusterLabelInformer.Lister(),
		cloudResource: resources.NewCsClusterLabels(cloud),
	}
}

// SyncWithCloud kicks of the Sync() function, should be started only after
// Informer caches we are about to use are synced
func (c *ClusterLabelSyncController) SyncWithCloud(stopCh <-chan struct{}) error {
	return c.syncWithCloud(c.doSync, stopCh)
}

func (c *ClusterLabelSyncController) doSync() {
	log.Debug("Sync ClusterLabels")
	// makes a request to containership api and write results to the resource's cache
	err := c.cloudResource.Sync()
	if err != nil {
		log.Error("ClusterLabels failed to sync: ", err.Error())
		return
	}

	// write the cloud items by ID so we can easily see if anything needs
	// to be deleted
	cloudCacheByID := make(map[string]interface{}, 0)

	for _, cloudItem := range c.cloudResource.Cache() {
		cloudCacheByID[cloudItem.ID] = cloudItem

		// Try to find cloud item in CR cache
		item, err := c.informer.GetIndexer().ByIndex(tools.IndexByIDFunctionName, cloudItem.ID)
		if err == nil && len(item) == 0 {
			log.Debugf("Cloud ClusterLabel %s does not exist as CR - creating", cloudItem.ID)
			err = c.Create(cloudItem)
			if err != nil {
				log.Error("ClusterLabel Create failed: ", err.Error())
			}
			continue
		}

		clusterLabelCR := item[0]
		if equal, err := c.cloudResource.IsEqual(cloudItem, clusterLabelCR); err == nil && !equal {
			log.Debugf("Cloud ClusterLabel %s does not match CR - updating", cloudItem.ID)
			err = c.Update(cloudItem, clusterLabelCR)
			if err != nil {
				log.Error("ClusterLabel Update failed: ", err.Error())
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
			log.Debugf("CR ClusterLabel %s does not exist in cloud - deleting", u.Name)
			err = c.Delete(u.Namespace, u.Name)
			if err != nil {
				log.Error("ClusterLabel Delete failed: ", err.Error())
			}
		}
	}
}

// Create takes a clusterLabel spec in cache and creates the CRD
func (c *ClusterLabelSyncController) Create(l csv3.ClusterLabelSpec) error {
	clusterLabel, err := c.clientset.ContainershipV3().ClusterLabels(constants.ContainershipNamespace).Create(&csv3.ClusterLabel{
		ObjectMeta: metav1.ObjectMeta{
			Name: l.ID,
		},
		Spec: l,
	})
	if err != nil {
		return err
	}

	// We can only fire an event if the object was successfully created,
	// otherwise there's no reasonable object to attach to.
	c.recorder.Event(clusterLabel, corev1.EventTypeNormal, "SyncCreate",
		"Detected missing CR")

	return nil
}

// Update takes a clusterLabel spec and updates the associated ClusterLabel CR spec
// with the new values
func (c *ClusterLabelSyncController) Update(l csv3.ClusterLabelSpec, obj interface{}) error {
	clusterLabel, ok := obj.(*csv3.ClusterLabel)
	if !ok {
		return fmt.Errorf("Error trying to use a non ClusterLabel CR object to update a ClusterLabel CR")
	}

	c.recorder.Event(clusterLabel, corev1.EventTypeNormal, "ClusterLabelUpdate",
		"Detected change in Cloud, updating")

	pCopy := clusterLabel.DeepCopy()
	pCopy.Spec = l

	_, err := c.clientset.ContainershipV3().ClusterLabels(constants.ContainershipNamespace).Update(pCopy)

	if err != nil {
		c.recorder.Eventf(clusterLabel, corev1.EventTypeWarning, "ClusterLabelUpdateError",
			"Error updating: %s", err.Error())
	}

	return err
}

// Delete takes a name of the CR and deletes it
func (c *ClusterLabelSyncController) Delete(namespace, name string) error {
	return c.clientset.ContainershipV3().ClusterLabels(namespace).Delete(name, &metav1.DeleteOptions{})
}
