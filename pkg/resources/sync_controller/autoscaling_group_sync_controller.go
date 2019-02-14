package synccontroller

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"

	cerebralv1alpha1 "github.com/containership/cerebral/pkg/apis/cerebral.containership.io/v1alpha1"

	cerebral "github.com/containership/cerebral/pkg/client/clientset/versioned"
	cerebralinformers "github.com/containership/cerebral/pkg/client/informers/externalversions"
	cerebrallisters "github.com/containership/cerebral/pkg/client/listers/cerebral.containership.io/v1alpha1"

	cscloud "github.com/containership/csctl/cloud"

	"github.com/containership/cluster-manager/pkg/log"
	"github.com/containership/cluster-manager/pkg/resources"
	"github.com/containership/cluster-manager/pkg/tools"
)

// AutoscalingGroupSyncController is the implementation for syncing AutoscalingGroup CRDs
type AutoscalingGroupSyncController struct {
	*syncController

	cerebralclientset cerebral.Interface
	lister            cerebrallisters.AutoscalingGroupLister
	cloudResource     *resources.CsAutoscalingGroups
}

const (
	autoscalingGroupSyncControllerName = "AutoscalingGroupSyncController"
)

// NewAutoscalingGroupController returns a AutoscalingGroupSyncController that
// will be in control of pulling from cloud comparing to the CR cache and
// modifying based on those compares
func NewAutoscalingGroupController(kubeclientset kubernetes.Interface, clientset cerebral.Interface, cerebralInformerFactory cerebralinformers.SharedInformerFactory, cloud cscloud.Interface) *AutoscalingGroupSyncController {
	autoscalingGroupInformer := cerebralInformerFactory.Cerebral().V1alpha1().AutoscalingGroups()

	autoscalingGroupInformer.Informer().AddIndexers(tools.IndexByIDKeyFun())

	return &AutoscalingGroupSyncController{
		syncController: &syncController{
			name:     autoscalingGroupSyncControllerName,
			synced:   autoscalingGroupInformer.Informer().HasSynced,
			informer: autoscalingGroupInformer.Informer(),
			recorder: tools.CreateAndStartRecorder(kubeclientset, autoscalingGroupSyncControllerName),
		},

		cerebralclientset: clientset,
		lister:            autoscalingGroupInformer.Lister(),
		cloudResource:     resources.NewCsAutoscalingGroups(cloud),
	}
}

// SyncWithCloud kicks of the Sync() function, should be started only after the
// informer caches we are about to use are synced
func (c *AutoscalingGroupSyncController) SyncWithCloud(stopCh <-chan struct{}) error {
	return c.syncWithCloud(c.doSync, stopCh)
}

func (c *AutoscalingGroupSyncController) doSync() {
	log.Debug("Sync AutoscalingGroups")
	// makes a request to containership api and writes results to the resource's cache
	err := c.cloudResource.Sync()
	if err != nil {
		log.Error("AutoscalingGroups failed to sync: ", err.Error())
		return
	}

	// write the cloud items by ID so we can easily see if anything needs
	// to be deleted
	cloudCacheByID := make(map[string]interface{}, 0)

	for _, cloudItem := range c.cloudResource.Cache() {
		cloudCacheByID[cloudItem.Name] = cloudItem

		// Try to find cloud item in CR cache
		item, err := c.informer.GetIndexer().ByIndex("byID", cloudItem.Name)
		if err == nil && len(item) == 0 {
			log.Debugf("Cloud AutoscalingGroup %s does not exist as CR - creating", cloudItem.Name)
			err = c.Create(cloudItem)
			if err != nil {
				log.Error("AutoscalingGroup Create failed: ", err.Error())
			}
			continue
		}

		autoscalingGroupCR := item[0]
		if equal, err := c.cloudResource.IsEqual(cloudItem.Spec, autoscalingGroupCR); err == nil && !equal {
			log.Debugf("Cloud AutoscalingGroup %s does not match CR - updating", cloudItem.Name)
			err = c.Update(cloudItem, autoscalingGroupCR)
			if err != nil {
				log.Error("AutoscalingGroup Update failed: ", err.Error())
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
	for _, ag := range allCRs {
		if _, exists := cloudCacheByID[ag.Name]; !exists {
			log.Debugf("CR AutoscalingGroup %s does not exist in cloud - deleting", ag.Name)
			err = c.Delete(ag.Name)
			if err != nil {
				log.Error("AutoscalingGroup Delete failed: ", err.Error())
			}
		}
	}
}

// Create takes an AutoscalingGroup in cache and creates the CR
func (c *AutoscalingGroupSyncController) Create(ag cerebralv1alpha1.AutoscalingGroup) error {
	autoscalingGroup, err := c.cerebralclientset.Cerebral().AutoscalingGroups().Create(&ag)
	if err != nil {
		return err
	}

	// We can only fire an event if the object was successfully created,
	// otherwise there's no reasonable object to attach to.
	c.recorder.Event(autoscalingGroup, corev1.EventTypeNormal, "SyncCreate",
		"Detected missing CR")

	return nil
}

// Update takes an AutoscalingGroup and updates the associated AutoscalingGroup CR spec
// with the new values
func (c *AutoscalingGroupSyncController) Update(updatedAG cerebralv1alpha1.AutoscalingGroup, obj interface{}) error {
	autoscalingGroup, ok := obj.(*cerebralv1alpha1.AutoscalingGroup)
	if !ok {
		return fmt.Errorf("Error trying to use a non AutoscalingGroup CR object to update a AutoscalingGroup CR")
	}

	c.recorder.Event(autoscalingGroup, corev1.EventTypeNormal, "AutoscalingGroupUpdate",
		"Detected change in Cloud, updating")

	pCopy := autoscalingGroup.DeepCopy()
	pCopy.Spec = updatedAG.Spec

	_, err := c.cerebralclientset.Cerebral().AutoscalingGroups().Update(pCopy)

	if err != nil {
		c.recorder.Eventf(autoscalingGroup, corev1.EventTypeWarning, "AutoscalingGroupUpdateError",
			"Error updating: %s", err.Error())
	}

	return err
}

// Delete takes the name of the CR and deletes it
func (c *AutoscalingGroupSyncController) Delete(name string) error {
	return c.cerebralclientset.Cerebral().AutoscalingGroups().Delete(name, &metav1.DeleteOptions{})
}
