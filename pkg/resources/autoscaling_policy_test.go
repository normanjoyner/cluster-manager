package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cerebralv1alpha1 "github.com/containership/cerebral/pkg/apis/cerebral.containership.io/v1alpha1"
)

var autoscalingPolicyBytes = []byte(`[{
	"name": "cpu-autoscaling-policy",
	"policy": {
		"scale_up": {
			"threshold": 0.9,
			"comparison_operator": ">=",
			"adjustment_type": "absolute",
			"adjustment_value": 1
		},
		"scale_down": {
			"threshold": 0.3,
			"comparison_operator": "<=",
			"adjustment_type": "absolute",
			"adjustment_value": 1
		}
	},
	"metrics_backend": "035d5705-f8c8-4878-b7b5-0b3309e8d4e9",
	"metric": "CPU",
	"metric_configuration": {
		"aggregation": "avg"
	},
	"poll_interval": 1234,
	"sample_period": 3000
}]`)

var spc1 = cerebralv1alpha1.ScalingPolicyConfiguration{
	Threshold:          0.8,
	ComparisonOperator: ">",
	AdjustmentType:     "absolute",
	AdjustmentValue:    2,
}

var spc2 = cerebralv1alpha1.ScalingPolicyConfiguration{
	Threshold:          0.8,
	ComparisonOperator: ">",
	AdjustmentType:     "absolute",
	AdjustmentValue:    1,
}

func TestScalingPolicyConfigurationIsEqual(t *testing.T) {
	result := scalingPolicyConfigurationIsEqual(spc1, spc1)
	assert.True(t, result)

	result = scalingPolicyConfigurationIsEqual(spc1, spc2)
	assert.False(t, result)
}

var sp1 = cerebralv1alpha1.ScalingPolicy{
	ScaleUp:   spc1,
	ScaleDown: spc1,
}

var sp2 = cerebralv1alpha1.ScalingPolicy{
	ScaleUp:   spc2,
	ScaleDown: spc2,
}

func TestScalingPolicyIsEqual(t *testing.T) {
	result := scalingPolicyIsEqual(sp1, sp1)
	assert.True(t, result)

	result = scalingPolicyIsEqual(sp1, sp2)
	assert.False(t, result)
}

func TestUnmarshalToCache(t *testing.T) {
	ap := NewCsAutoscalingPolicies()

	err := ap.UnmarshalToCache(nil)
	assert.Error(t, err)

	err = ap.UnmarshalToCache(autoscalingPolicyBytes)
	assert.Nil(t, err)
}

func TestAutoscalingGroupCache(t *testing.T) {
	ap := NewCsAutoscalingPolicies()
	ap.UnmarshalToCache(autoscalingPolicyBytes)
	c := ap.Cache()

	assert.Equal(t, ap.cache, c)

	v, found := c[0].Spec.MetricConfiguration["aggregation"]
	assert.True(t, found)
	assert.Equal(t, "avg", v)
}

// Testing IsEqual function
var cloud = cerebralv1alpha1.AutoscalingPolicySpec{
	Metric: "CPU",
}

var cloudPolicyChange = cerebralv1alpha1.AutoscalingPolicySpec{
	Metric:        "CPU",
	ScalingPolicy: sp2,
}

var cloudChange = cerebralv1alpha1.AutoscalingPolicySpec{
	Metric: "Memory",
}

var cacheObj = &cerebralv1alpha1.AutoscalingPolicy{
	ObjectMeta: metav1.ObjectMeta{
		Name: "asp1",
	},
	Spec: cerebralv1alpha1.AutoscalingPolicySpec{
		Metric: "CPU",
	},
}

func TestAutoscalingPolicyIsEqual(t *testing.T) {
	ap := NewCsAutoscalingPolicies()

	result, err := ap.IsEqual(cloud, cacheObj)
	assert.Nil(t, err)
	assert.True(t, result)

	result, err = ap.IsEqual(cloudChange, cacheObj)
	assert.Nil(t, err)
	assert.False(t, result)

	result, err = ap.IsEqual(cloudPolicyChange, cacheObj)
	assert.Nil(t, err)
	assert.False(t, result)

	_, err = ap.IsEqual(cacheObj, cacheObj)
	assert.Error(t, err)

	_, err = ap.IsEqual(cloud, cloud)
	assert.Error(t, err)
}
