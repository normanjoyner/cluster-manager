package resources

import (
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/containership/cluster-manager/pkg/request"

	cerebralv1alpha1 "github.com/containership/cerebral/pkg/apis/cerebral.containership.io/v1alpha1"
)

// CsAutoscalingPolicies defines the Containership Cloud AutoscalingEngines resource
type CsAutoscalingPolicies struct {
	cloudResource
	cache []cerebralv1alpha1.AutoscalingPolicy
}

// CloudAPIAutoscalingPolicy is the spec for a autoscaling group
type CloudAPIAutoscalingPolicy struct {
	ID                  string                `json:"id"`
	MetricsBackend      string                `json:"metrics_backend"`
	Metric              string                `json:"metric"`
	MetricConfiguration map[string]string     `json:"metric_configuration"`
	ScalingPolicy       CloudAPIScalingPolicy `json:"scaling_policy"`
	PollInterval        int                   `json:"poll_interval"`
	SamplePeriod        int                   `json:"sample_period"`
}

// CloudAPIScalingPolicy holds the policy configurations for scaling up and down
type CloudAPIScalingPolicy struct {
	ScaleUp   CloudAPIScalingPolicyConfiguration `json:"scale_up"`
	ScaleDown CloudAPIScalingPolicyConfiguration `json:"scale_down"`
}

// A CloudAPIScalingPolicyConfiguration defines the criterion for triggering a scale event
type CloudAPIScalingPolicyConfiguration struct {
	Threshold          float64 `json:"threshold"`
	ComparisonOperator string  `json:"comparison_operator"`
	AdjustmentType     string  `json:"adjustment_type"`
	AdjustmentValue    int     `json:"adjustment_value"`
}

// NewCsAutoscalingPolicies constructs a new CsAutoscalingPolicies
func NewCsAutoscalingPolicies() *CsAutoscalingPolicies {
	return &CsAutoscalingPolicies{
		cloudResource: cloudResource{
			endpoint: "/organizations/{{.OrganizationID}}/clusters/{{.ClusterID}}/autoscaling-policies",
			service:  request.CloudServiceProvision,
		},
		cache: make([]cerebralv1alpha1.AutoscalingPolicy, 0),
	}
}

// UnmarshalToCache take the json returned from containership api
// and writes it to CsAutoscalingPolicies cache
func (us *CsAutoscalingPolicies) UnmarshalToCache(bytes []byte) error {
	cloudAutoscalingPolicies := make([]CloudAPIAutoscalingPolicy, 0)
	err := json.Unmarshal(bytes, &cloudAutoscalingPolicies)
	if err != nil {
		return err
	}

	coerceCloudAPITypeToCerebral := make([]cerebralv1alpha1.AutoscalingPolicy, len(cloudAutoscalingPolicies))
	for i, cap := range cloudAutoscalingPolicies {

		// This converts the autoscaling policy object returned from the containership
		// API into the Cerebral AutoscalingPolicy type. This object will then get written
		// to the associated CR in camel case. In order to change to the Cerebral
		// AutoscalingPolicy type you need to reference each value to convert, since
		// the underlying types and structure is not consistent between the API representation
		// of the object and the Cerebral representation of the object.
		ae := cerebralv1alpha1.AutoscalingPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name: cap.ID,
			},
			Spec: cerebralv1alpha1.AutoscalingPolicySpec{
				MetricsBackend:      cap.MetricsBackend,
				Metric:              cap.Metric,
				MetricConfiguration: cap.MetricConfiguration,
				PollInterval:        cap.PollInterval,
				SamplePeriod:        cap.SamplePeriod,
				ScalingPolicy: cerebralv1alpha1.ScalingPolicy{
					ScaleUp: cerebralv1alpha1.ScalingPolicyConfiguration{
						Threshold:          cap.ScalingPolicy.ScaleUp.Threshold,
						ComparisonOperator: cap.ScalingPolicy.ScaleUp.ComparisonOperator,
						AdjustmentType:     cap.ScalingPolicy.ScaleUp.AdjustmentType,
						AdjustmentValue:    cap.ScalingPolicy.ScaleUp.AdjustmentValue,
					},
					ScaleDown: cerebralv1alpha1.ScalingPolicyConfiguration{
						Threshold:          cap.ScalingPolicy.ScaleDown.Threshold,
						ComparisonOperator: cap.ScalingPolicy.ScaleDown.ComparisonOperator,
						AdjustmentType:     cap.ScalingPolicy.ScaleDown.AdjustmentType,
						AdjustmentValue:    cap.ScalingPolicy.ScaleDown.AdjustmentValue,
					},
				},
			},
		}

		coerceCloudAPITypeToCerebral[i] = ae
	}

	us.cache = coerceCloudAPITypeToCerebral

	return nil
}

// Cache returns the autoscalingEngines cache
func (us *CsAutoscalingPolicies) Cache() []cerebralv1alpha1.AutoscalingPolicy {
	return us.cache
}

// IsEqual compares a AutoscalingEngineSpec to another AutoscalingEngine
// new, cache
func (us *CsAutoscalingPolicies) IsEqual(specObj interface{}, parentSpecObj interface{}) (bool, error) {
	spec, ok := specObj.(cerebralv1alpha1.AutoscalingPolicySpec)
	if !ok {
		return false, fmt.Errorf("The object is not of type AutoscalingPolicySpec")
	}

	autoscalingPolicy, ok := parentSpecObj.(*cerebralv1alpha1.AutoscalingPolicy)
	if !ok {
		return false, fmt.Errorf("The object is not of type AutoscalingPolicy")
	}

	equal := autoscalingPolicy.Spec.MetricsBackend == spec.MetricsBackend &&
		autoscalingPolicy.Spec.Metric == spec.Metric &&
		autoscalingPolicy.Spec.PollInterval == spec.PollInterval &&
		autoscalingPolicy.Spec.SamplePeriod == spec.SamplePeriod

	if !equal {
		return false, nil
	}

	equal = scalingPolicyIsEqual(spec.ScalingPolicy, autoscalingPolicy.Spec.ScalingPolicy)

	return equal, nil
}

func scalingPolicyIsEqual(cloudPolicies, cachePolicies cerebralv1alpha1.ScalingPolicy) bool {
	return scalingPolicyConfigurationIsEqual(cloudPolicies.ScaleUp, cachePolicies.ScaleUp) &&
		scalingPolicyConfigurationIsEqual(cloudPolicies.ScaleDown, cachePolicies.ScaleDown)
}

func scalingPolicyConfigurationIsEqual(cloudPolicyConfiguration, cachePolicyConfiguration cerebralv1alpha1.ScalingPolicyConfiguration) bool {
	return cloudPolicyConfiguration.Threshold == cachePolicyConfiguration.Threshold &&
		cloudPolicyConfiguration.ComparisonOperator == cachePolicyConfiguration.ComparisonOperator &&
		cloudPolicyConfiguration.AdjustmentType == cachePolicyConfiguration.AdjustmentType &&
		cloudPolicyConfiguration.AdjustmentValue == cachePolicyConfiguration.AdjustmentValue
}
