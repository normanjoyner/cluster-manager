package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cerebralv1alpha1 "github.com/containership/cerebral/pkg/apis/cerebral.containership.io/v1alpha1"
)

var autoscalingPolicyBytes = []byte(`[{
	"id": "1234",
	"name": "cpu-autoscaling-policy",
	"scaling_policy": {
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

var spc1 = &cerebralv1alpha1.ScalingPolicyConfiguration{
	Threshold:          0.8,
	ComparisonOperator: ">",
	AdjustmentType:     "absolute",
	AdjustmentValue:    2,
}

var spc2 = &cerebralv1alpha1.ScalingPolicyConfiguration{
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

	result = scalingPolicyConfigurationIsEqual(nil, nil)
	assert.True(t, result)

	result = scalingPolicyConfigurationIsEqual(spc1, nil)
	assert.False(t, result)

	result = scalingPolicyConfigurationIsEqual(nil, spc1)
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

var bytesToAutoscalingPolicy = cerebralv1alpha1.AutoscalingPolicy{
	ObjectMeta: metav1.ObjectMeta{
		Name: "1234",
	},
	Spec: cerebralv1alpha1.AutoscalingPolicySpec{
		MetricsBackend: "035d5705-f8c8-4878-b7b5-0b3309e8d4e9",
		Metric:         "CPU",
		MetricConfiguration: map[string]string{
			"aggregation": "avg",
		},
		PollInterval: 1234,
		SamplePeriod: 3000,
		ScalingPolicy: cerebralv1alpha1.ScalingPolicy{
			ScaleUp: &cerebralv1alpha1.ScalingPolicyConfiguration{
				Threshold:          0.9,
				ComparisonOperator: ">=",
				AdjustmentType:     "absolute",
				AdjustmentValue:    1,
			},
			ScaleDown: &cerebralv1alpha1.ScalingPolicyConfiguration{
				Threshold:          0.3,
				ComparisonOperator: "<=",
				AdjustmentType:     "absolute",
				AdjustmentValue:    1,
			},
		},
	},
}

func TestUnmarshalAutoscalingPolicyToCache(t *testing.T) {
	ap := NewCsAutoscalingPolicies()

	err := ap.UnmarshalToCache(nil)
	assert.Error(t, err)

	err = ap.UnmarshalToCache(autoscalingPolicyBytes)
	assert.Nil(t, err)
	assert.Equal(t, []cerebralv1alpha1.AutoscalingPolicy{bytesToAutoscalingPolicy}, ap.cache)
}

func TestAutoscalingPolicyCache(t *testing.T) {
	ap := NewCsAutoscalingPolicies()
	ap.UnmarshalToCache(autoscalingPolicyBytes)
	c := ap.Cache()

	assert.Equal(t, ap.cache, c)

	v, found := c[0].Spec.MetricConfiguration["aggregation"]
	assert.True(t, found)
	assert.Equal(t, "avg", v)
}

// Testing IsEqual function
var cloudAP = cerebralv1alpha1.AutoscalingPolicySpec{
	Metric: "CPU",
}

var cloudAPPolicyChange = cerebralv1alpha1.AutoscalingPolicySpec{
	Metric:        "CPU",
	ScalingPolicy: sp2,
}

var cloudAPChange = cerebralv1alpha1.AutoscalingPolicySpec{
	Metric: "Memory",
}

var autoscalingPolicyCacheObj = &cerebralv1alpha1.AutoscalingPolicy{
	ObjectMeta: metav1.ObjectMeta{
		Name: "asp1",
	},
	Spec: cerebralv1alpha1.AutoscalingPolicySpec{
		Metric: "CPU",
	},
}

func TestAutoscalingPolicyIsEqual(t *testing.T) {
	ap := NewCsAutoscalingPolicies()

	result, err := ap.IsEqual(cloudAP, autoscalingPolicyCacheObj)
	assert.Nil(t, err)
	assert.True(t, result)

	result, err = ap.IsEqual(cloudAPChange, autoscalingPolicyCacheObj)
	assert.Nil(t, err)
	assert.False(t, result)

	result, err = ap.IsEqual(cloudAPPolicyChange, autoscalingPolicyCacheObj)
	assert.Nil(t, err)
	assert.False(t, result)

	_, err = ap.IsEqual(autoscalingPolicyCacheObj, autoscalingPolicyCacheObj)
	assert.Error(t, err)

	_, err = ap.IsEqual(cloudAP, cloudAP)
	assert.Error(t, err)
}

var spConverstion = &CloudAPIScalingPolicyConfiguration{
	Threshold:          0.8,
	ComparisonOperator: ">",
	AdjustmentType:     "percent",
	AdjustmentValue:    0.1,
}

func TestConvertScalingPolicy(t *testing.T) {
	p := convertScalingPolicy(nil)
	assert.Nil(t, p)

	empty := &CloudAPIScalingPolicyConfiguration{}
	p = convertScalingPolicy(empty)
	assert.Nil(t, p)

	p = convertScalingPolicy(spConverstion)
	assert.Equal(t, spConverstion.Threshold, p.Threshold)
	assert.Equal(t, spConverstion.ComparisonOperator, p.ComparisonOperator)
	assert.Equal(t, spConverstion.AdjustmentType, p.AdjustmentType)
	assert.Equal(t, spConverstion.AdjustmentValue, p.AdjustmentValue)
}
