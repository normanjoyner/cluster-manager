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

	"github.com/containership/cluster-manager/pkg/log"
	"github.com/containership/cluster-manager/pkg/resources"
	"github.com/containership/cluster-manager/pkg/tools"
)

// AutoscalingPolicySyncController is the implementation for syncing AutoscalingPolicy CRDs
type AutoscalingPolicySyncController struct {
	*syncController

	cerebralclientset cerebral.Interface
	cloudResource     *resources.CsAutoscalingPolicies
	lister            cerebrallisters.AutoscalingPolicyLister
}

const (
	autoscalingPolicySyncControllerName = "AutoscalingPolicySyncController"
)

// NewAutoscalingPolicyController returns a AutoscalingPolicySyncController that
// will be in control of pulling from cloud comparing to the CR cache and
// modifying based on those compares
func NewAutoscalingPolicyController(kubeclientset kubernetes.Interface, clientset cerebral.Interface, cerebralInformerFactory cerebralinformers.SharedInformerFactory) *AutoscalingPolicySyncController {
	autoscalingPolicyInformer := cerebralInformerFactory.Cerebral().V1alpha1().AutoscalingPolicies()

	autoscalingPolicyInformer.Informer().AddIndexers(tools.IndexByIDKeyFun())

	return &AutoscalingPolicySyncController{
		syncController: &syncController{
			name:     autoscalingPolicySyncControllerName,
			synced:   autoscalingPolicyInformer.Informer().HasSynced,
			informer: autoscalingPolicyInformer.Informer(),
			recorder: tools.CreateAndStartRecorder(kubeclientset, autoscalingPolicySyncControllerName),
		},

		cerebralclientset: clientset,
		lister:            autoscalingPolicyInformer.Lister(),
		cloudResource:     resources.NewCsAutoscalingPolicies(),
	}
}

// SyncWithCloud kicks of the Sync() function, should be started only after
// Informer caches are synced
func (c *AutoscalingPolicySyncController) SyncWithCloud(stopCh <-chan struct{}) error {
	return c.syncWithCloud(c.doSync, stopCh)
}

func (c *AutoscalingPolicySyncController) doSync() {
	log.Debug("Sync AutoscalingPolicies")
	// makes a request to containership api and write results to the resource's cache
	err := resources.Sync(c.cloudResource)
	if err != nil {
		log.Error("AutoscalingPolicies failed to sync: ", err.Error())
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
			log.Debugf("Cloud AutoscalingPolicy %s does not exist as CR - creating", cloudItem.Name)
			err = c.Create(cloudItem)
			if err != nil {
				log.Error("AutoscalingPolicy Create failed: ", err.Error())
			}
			continue
		}

		autoscalingPolicyCR := item[0]
		if equal, err := c.cloudResource.IsEqual(cloudItem.Spec, autoscalingPolicyCR); err == nil && !equal {
			log.Debugf("Cloud AutoscalingPolicy %s does not match CR - updating", cloudItem.Name)
			err = c.Update(cloudItem, autoscalingPolicyCR)
			if err != nil {
				log.Error("AutoscalingPolicy Update failed: ", err.Error())
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
			log.Debugf("CR AutoscalingPolicy %s does not exist in cloud - deleting", ag.Name)
			err = c.Delete(ag.Name)
			if err != nil {
				log.Error("AutoscalingPolicy Delete failed: ", err.Error())
			}
		}
	}
}

// Create takes a AutoscalingPolicy in cache and creates the CR
func (c *AutoscalingPolicySyncController) Create(ag cerebralv1alpha1.AutoscalingPolicy) error {
	autoscalingPolicy, err := c.cerebralclientset.Cerebral().AutoscalingPolicies().Create(&ag)
	if err != nil {
		return err
	}

	// We can only fire an event if the object was successfully created,
	// otherwise there's no reasonable object to attach to.
	c.recorder.Event(autoscalingPolicy, corev1.EventTypeNormal, "SyncCreate",
		"Detected missing CR")

	return nil
}

// Update takes an AutoscalingPolicy spec and updates the associated AutoscalingPolicy CR spec
// with the new values
func (c *AutoscalingPolicySyncController) Update(updatedAG cerebralv1alpha1.AutoscalingPolicy, obj interface{}) error {
	autoscalingPolicy, ok := obj.(*cerebralv1alpha1.AutoscalingPolicy)
	if !ok {
		return fmt.Errorf("Error trying to use a non AutoscalingPolicy CR object to update a AutoscalingPolicy CR")
	}

	c.recorder.Event(autoscalingPolicy, corev1.EventTypeNormal, "AutoscalingPolicyUpdate",
		"Detected change in Cloud, updating")

	pCopy := autoscalingPolicy.DeepCopy()
	pCopy.Spec = updatedAG.Spec

	_, err := c.cerebralclientset.Cerebral().AutoscalingPolicies().Update(pCopy)

	if err != nil {
		c.recorder.Eventf(autoscalingPolicy, corev1.EventTypeWarning, "AutoscalingPolicyUpdateError",
			"Error updating: %s", err.Error())
	}

	return err
}

// Delete takes a name or the CRD and deletes it
func (c *AutoscalingPolicySyncController) Delete(name string) error {
	return c.cerebralclientset.Cerebral().AutoscalingPolicies().Delete(name, &metav1.DeleteOptions{})
}
