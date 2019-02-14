package resources

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cerebralv1alpha1 "github.com/containership/cerebral/pkg/apis/cerebral.containership.io/v1alpha1"
	"github.com/containership/csctl/cloud/provision/types"
)

var autoscalingGroupBytesDisabled = []byte(`[{
	"id": "1234",
	"autoscaling": {
		"node_selector": {
			"key": "value"
		},
		"enabled" : false,
		"policies": ["policies"],
		"engine": "containership",
		"cooldown_period": 600,
		"suspended": true,
		"min_nodes": 1,
		"max_nodes": 3,
		"scaling_strategy": {
			"scale_up": "random",
			"scale_down": "leastPods"
		}
	}
}]`)

type strategiesEqualTest struct {
	s1 *cerebralv1alpha1.ScalingStrategy
	s2 *cerebralv1alpha1.ScalingStrategy

	expected bool
	message  string
}

var strategiesEqualTests = []strategiesEqualTest{
	{
		s1:       nil,
		s2:       nil,
		expected: true,
		message:  "nil strategy == nil strategy",
	},
	{
		s1:       &cerebralv1alpha1.ScalingStrategy{},
		s2:       nil,
		expected: false,
		message:  "empty strategy != nil strategy",
	},
	{
		s1:       nil,
		s2:       &cerebralv1alpha1.ScalingStrategy{},
		expected: false,
		message:  "nil strategy != empty strategy",
	},
	{
		s1:       &cerebralv1alpha1.ScalingStrategy{},
		s2:       &cerebralv1alpha1.ScalingStrategy{},
		expected: true,
		message:  "empty strategy == empty strategy",
	},
	{
		s1: &cerebralv1alpha1.ScalingStrategy{
			ScaleUp: "up-strategy",
		},
		s2:       &cerebralv1alpha1.ScalingStrategy{},
		expected: false,
		message:  "non-empty scale up strategy only != empty scale up strategy",
	},
	{
		s1: &cerebralv1alpha1.ScalingStrategy{
			ScaleDown: "up-strategy",
		},
		s2:       &cerebralv1alpha1.ScalingStrategy{},
		expected: false,
		message:  "non-empty scale up strategy only != empty scale up strategy",
	},
	{
		s1: &cerebralv1alpha1.ScalingStrategy{
			ScaleUp: "up-strategy",
		},
		s2: &cerebralv1alpha1.ScalingStrategy{
			ScaleUp: "up-strategy",
		},
		expected: true,
		message:  "scale up strategy only == scale up strategy only",
	},
	{
		s1: &cerebralv1alpha1.ScalingStrategy{
			ScaleUp: "up-strategy",
		},
		s2: &cerebralv1alpha1.ScalingStrategy{
			ScaleUp: "different-up-strategy",
		},
		expected: false,
		message:  "scale up strategy only != different scale up strategy only",
	},
	{
		s1: &cerebralv1alpha1.ScalingStrategy{
			ScaleDown: "down-strategy",
		},
		s2: &cerebralv1alpha1.ScalingStrategy{
			ScaleDown: "down-strategy",
		},
		expected: true,
		message:  "scale down strategy only == scale down strategy only",
	},
	{
		s1: &cerebralv1alpha1.ScalingStrategy{
			ScaleDown: "down-strategy",
		},
		s2: &cerebralv1alpha1.ScalingStrategy{
			ScaleDown: "different-down-strategy",
		},
		expected: false,
		message:  "scale down strategy only != different scale down strategy only",
	},
	{
		s1: &cerebralv1alpha1.ScalingStrategy{
			ScaleUp:   "up-strategy",
			ScaleDown: "down-strategy",
		},
		s2: &cerebralv1alpha1.ScalingStrategy{
			ScaleUp:   "up-strategy",
			ScaleDown: "down-strategy",
		},
		expected: true,
		message:  "scale strategies the same",
	},
}

func TestScalingStrategiesAreEqual(t *testing.T) {
	for _, test := range strategiesEqualTests {
		equal := scalingStrategiesAreEqual(test.s1, test.s2)
		assert.Equal(t, test.expected, equal, test.message)
	}
}

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
	ag := NewCsAutoscalingGroups(nil)

	err := ag.UnmarshalToCache(nil)
	assert.Error(t, err)

	err = ag.UnmarshalToCache(autoscalingGroupBytesDisabled)
	assert.Nil(t, err)
}

func TestAutoscalingGroupCache(t *testing.T) {
	ag := NewCsAutoscalingGroups(nil)
	ag.UnmarshalToCache(autoscalingGroupBytesDisabled)
	c := ag.Cache()

	assert.Equal(t, ag.cache, c)
}

var autoscalingGroupBytes = []byte(`{
	"id": "1234",
	"autoscaling": {
		"node_selector": {
			"key": "value"
		},
		"enabled" : true,
		"policies": ["policies"],
		"engine": "containership",
		"cooldown_period": 600,
		"suspended": true,
		"min_nodes": 1,
		"max_nodes": 3,
		"scaling_strategy": {
			"scale_up": "random",
			"scale_down": "leastPods"
		}
	}
}`)

var cerebralAutoscalingGroup = cerebralv1alpha1.AutoscalingGroup{
	ObjectMeta: metav1.ObjectMeta{
		Name: "1234",
	},
	Spec: cerebralv1alpha1.AutoscalingGroupSpec{
		NodeSelector: map[string]string{
			"key": "value",
		},
		Policies:       []string{"policy-id"},
		Engine:         "containership",
		CooldownPeriod: 600,
		Suspended:      true,
		MinNodes:       1,
		MaxNodes:       3,
		ScalingStrategy: &cerebralv1alpha1.ScalingStrategy{
			ScaleUp:   "random",
			ScaleDown: "leastPods",
		},
	},
}

func TestTransformAPINodePoolToCerebralASG(t *testing.T) {
	var np NodePool
	err := json.Unmarshal(autoscalingGroupBytes, &np)
	assert.Nil(t, err)
	policyIDs := []string{"policy-id"}

	ag := transformAPINodePoolToCerebralASG(np, policyIDs)
	assert.Equal(t, cerebralAutoscalingGroup, ag)
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
	ag := NewCsAutoscalingGroups(nil)

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
	policies := getAutoscalingGroupPoliciesIDs(autoscalingPolicyBytes)
	assert.Equal(t, "1234", policies[0])

	policies = getAutoscalingGroupPoliciesIDs([]types.AutoscalingPolicy{})
	assert.Equal(t, 0, len(policies))

	policies = getAutoscalingGroupPoliciesIDs(nil)
	assert.Equal(t, 0, len(policies))
}
