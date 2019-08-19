package synccontroller

import (
	"fmt"

	cscloud "github.com/containership/csctl/cloud"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"

	csv3 "github.com/containership/cluster-manager/pkg/apis/provision.containership.io/v3"
	csclientset "github.com/containership/cluster-manager/pkg/client/clientset/versioned"
	csinformers "github.com/containership/cluster-manager/pkg/client/informers/externalversions"
	cslisters "github.com/containership/cluster-manager/pkg/client/listers/provision.containership.io/v3"

	"github.com/containership/cluster-manager/pkg/constants"
	"github.com/containership/cluster-manager/pkg/log"
	"github.com/containership/cluster-manager/pkg/resources"
	"github.com/containership/cluster-manager/pkg/tools"
)

// NodePoolLabelSyncController is the implementation for syncing NodePoolLabel CRDs
type NodePoolLabelSyncController struct {
	*syncController

	lister        cslisters.NodePoolLabelLister
	cloudResource *resources.CsNodePoolLabels
}

const (
	nodePoolLabelSyncControllerName = "NodePoolLabelSyncController"
)

// NewNodePoolLabel returns a NodePoolLabelSyncController that will be in control of pulling from cloud
// comparing to the CRD cache and modifying based on those compares
func NewNodePoolLabel(kubeclientset kubernetes.Interface, clientset csclientset.Interface, csInformerFactory csinformers.SharedInformerFactory, cloud cscloud.Interface) *NodePoolLabelSyncController {
	nodePoolLabelInformer := csInformerFactory.ContainershipProvision().V3().NodePoolLabels()

	nodePoolLabelInformer.Informer().AddIndexers(tools.IndexByIDKeyFun())

	return &NodePoolLabelSyncController{
		syncController: &syncController{
			name:      nodePoolLabelSyncControllerName,
			clientset: clientset,
			synced:    nodePoolLabelInformer.Informer().HasSynced,
			informer:  nodePoolLabelInformer.Informer(),
			recorder:  tools.CreateAndStartRecorder(kubeclientset, nodePoolLabelSyncControllerName),
		},

		lister:        nodePoolLabelInformer.Lister(),
		cloudResource: resources.NewCsNodePoolLabels(cloud),
	}
}

// SyncWithCloud kicks of the Sync() function, should be started only after
// Informer caches we are about to use are synced
func (c *NodePoolLabelSyncController) SyncWithCloud(stopCh <-chan struct{}) error {
	return c.syncWithCloud(c.doSync, stopCh)
}

func (c *NodePoolLabelSyncController) doSync() {
	log.Debug("Sync NodePoolLabels")
	// makes a request to containership api and write results to the resource's cache
	err := c.cloudResource.Sync()
	if err != nil {
		log.Error("NodePoolLabels failed to sync: ", err.Error())
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
			log.Debugf("Cloud NodePoolLabel %s does not exist as CR - creating", cloudItem.ID)
			err = c.Create(cloudItem)
			if err != nil {
				log.Error("NodePoolLabel Create failed: ", err.Error())
			}
			continue
		}

		nodePoolLabelCR := item[0]
		if equal, err := c.cloudResource.IsEqual(cloudItem, nodePoolLabelCR); err == nil && !equal {
			log.Debugf("Cloud NodePoolLabel %s does not match CR - updating", cloudItem.ID)
			err = c.Update(cloudItem, nodePoolLabelCR)
			if err != nil {
				log.Error("NodePoolLabel Update failed: ", err.Error())
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
			log.Debugf("CR NodePoolLabel %s does not exist in cloud - deleting", u.Name)
			err = c.Delete(u.Namespace, u.Name)
			if err != nil {
				log.Error("NodePoolLabel Delete failed: ", err.Error())
			}
		}
	}
}

// Create takes a nodePoolLabel spec in cache and creates the CRD
// The standard Containership node pool label is added so we can easily
// get only node pool labels for a certain pool later on
func (c *NodePoolLabelSyncController) Create(l csv3.NodePoolLabelSpec) error {
	nodePoolLabel, err := c.clientset.ContainershipProvisionV3().NodePoolLabels(constants.ContainershipNamespace).Create(&csv3.NodePoolLabel{
		ObjectMeta: metav1.ObjectMeta{
			Name: l.ID,
			Labels: map[string]string{
				constants.ContainershipNodePoolIDLabelKey: l.NodePoolID,
			},
		},
		Spec: l,
	})
	if err != nil {
		return err
	}

	// We can only fire an event if the object was successfully created,
	// otherwise there's no reasonable object to attach to.
	c.recorder.Event(nodePoolLabel, corev1.EventTypeNormal, "SyncCreate",
		"Detected missing CR")

	return nil
}

// Update takes a nodePoolLabel spec and updates the associated NodePoolLabel CR spec
// with the new values
func (c *NodePoolLabelSyncController) Update(l csv3.NodePoolLabelSpec, obj interface{}) error {
	nodePoolLabel, ok := obj.(*csv3.NodePoolLabel)
	if !ok {
		return fmt.Errorf("Error trying to use a non NodePoolLabel CR object to update a NodePoolLabel CR")
	}

	c.recorder.Event(nodePoolLabel, corev1.EventTypeNormal, "NodePoolLabelUpdate",
		"Detected change in Cloud, updating")

	pCopy := nodePoolLabel.DeepCopy()
	pCopy.Spec = l

	_, err := c.clientset.ContainershipProvisionV3().NodePoolLabels(constants.ContainershipNamespace).Update(pCopy)

	if err != nil {
		c.recorder.Eventf(nodePoolLabel, corev1.EventTypeWarning, "NodePoolLabelUpdateError",
			"Error updating: %s", err.Error())
	}

	return err
}

// Delete takes a name of the CR and deletes it
func (c *NodePoolLabelSyncController) Delete(namespace, name string) error {
	return c.clientset.ContainershipProvisionV3().NodePoolLabels(namespace).Delete(name, &metav1.DeleteOptions{})
}
