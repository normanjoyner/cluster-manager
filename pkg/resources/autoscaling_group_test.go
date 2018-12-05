package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cerebralv1alpha1 "github.com/containership/cerebral/pkg/apis/cerebral.containership.io/v1alpha1"
)

var autoscalingGroupBytes = []byte(`[{
	"id": "1234",
	"autoscaling": {
		"node_selector": {
			"key": "value"
		},
		"enabled" : false,
		"policies": ["policies"],
		"engine": "containership",
		"cooldown_period": 600,
		"suspend": false,
		"min_nodes": 1,
		"max_nodes": 3,
		"scaling_strategy": {
			"scale_up": "random",
			"scale_down": "leastPods"
		}
	}
}]`)

func TestPoliciesAreEqual(t *testing.T) {
	policies1 := []string{"policy-1", "policy-2", "policy-3"}
	policies2 := []string{"policy-1", "policy-2"}
	policies3 := []string{"policy-1", "policy-2", "policy-4"}

	result := policiesAreEqual(policies1, policies1)
	assert.True(t, result)
	// does not match different lengths
	result = policiesAreEqual(policies1, policies2)
	assert.False(t, result)
	// does not match same length
	result = policiesAreEqual(policies1, policies3)
	assert.False(t, result)
}

func TestUnmarshalToCache(t *testing.T) {
	ag := NewCsAutoscalingGroups()

	err := ag.UnmarshalToCache(nil)
	assert.Error(t, err)

	err = ag.UnmarshalToCache(autoscalingGroupBytes)
	assert.Nil(t, err)
}

func TestAutoscalingGroupCache(t *testing.T) {
	ag := NewCsAutoscalingGroups()
	// TODO: add actual object
	ag.UnmarshalToCache(autoscalingGroupBytes)
	c := ag.Cache()

	assert.Equal(t, ag.cache, c)
}

// Testing IsEqual function
var cloud = cerebralv1alpha1.AutoscalingGroupSpec{
	NodeSelector: map[string]string{
		"key": "value",
	},
	Engine: "containership",
}

var cloudNodeSelectorChange = cerebralv1alpha1.AutoscalingGroupSpec{
	NodeSelector: map[string]string{
		"key": "value2",
	},
	Engine: "containership",
}

var autoscalingGroupCloudChange = cerebralv1alpha1.AutoscalingGroupSpec{
	Engine: "newEngine",
}

var autoscalingGroupCacheObj = &cerebralv1alpha1.AutoscalingGroup{
	ObjectMeta: metav1.ObjectMeta{
		Name: "asg1",
	},
	Spec: cerebralv1alpha1.AutoscalingGroupSpec{
		NodeSelector: map[string]string{
			"key": "value",
		},
		Engine: "containership",
	},
}

func TestAGIsEqual(t *testing.T) {
	ag := NewCsAutoscalingGroups()

	result, err := ag.IsEqual(cloud, autoscalingGroupCacheObj)
	assert.Nil(t, err)
	assert.True(t, result)

	result, err = ag.IsEqual(autoscalingGroupCloudChange, autoscalingGroupCacheObj)
	assert.Nil(t, err)
	assert.False(t, result)

	result, err = ag.IsEqual(cloudNodeSelectorChange, autoscalingGroupCacheObj)
	assert.Nil(t, err)
	assert.False(t, result)
	// use AutoscalingGroup Type as first argument
	_, err = ag.IsEqual(autoscalingGroupCacheObj, autoscalingGroupCacheObj)
	assert.Error(t, err)
	// use AutoscalingGroupSpec Type as second argument
	_, err = ag.IsEqual(cloud, cloud)
	assert.Error(t, err)
}

func TestGetAutoscalingGroupPoliciesIDs(t *testing.T) {
	policies, err := getAutoscalingGroupPoliciesIDs(autoscalingPolicyBytes)
	assert.Nil(t, err)
	assert.Equal(t, "1234", policies[0])

	policies, err = getAutoscalingGroupPoliciesIDs([]byte("[]"))
	assert.Nil(t, err)
	assert.Equal(t, 0, len(policies))

	_, err = getAutoscalingGroupPoliciesIDs(nil)
	assert.Error(t, err)
}
