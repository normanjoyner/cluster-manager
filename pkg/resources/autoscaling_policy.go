package resources

import (
	"errors"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cscloud "github.com/containership/csctl/cloud"
	"github.com/containership/csctl/cloud/provision/types"

	"github.com/containership/cluster-manager/pkg/tools"

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
	ScaleUp   *CloudAPIScalingPolicyConfiguration `json:"scale_up"`
	ScaleDown *CloudAPIScalingPolicyConfiguration `json:"scale_down"`
}

// A CloudAPIScalingPolicyConfiguration defines the criterion for triggering a scale event
type CloudAPIScalingPolicyConfiguration struct {
	Threshold          float64 `json:"threshold"`
	ComparisonOperator string  `json:"comparison_operator"`
	AdjustmentType     string  `json:"adjustment_type"`
	AdjustmentValue    float64 `json:"adjustment_value"`
}

// NewCsAutoscalingPolicies constructs a new CsAutoscalingPolicies
func NewCsAutoscalingPolicies(cloud cscloud.Interface) *CsAutoscalingPolicies {
	cache := make([]cerebralv1alpha1.AutoscalingPolicy, 0)
	return &CsAutoscalingPolicies{
		newCloudResource(cloud),
		cache,
	}
}

// Sync implements the CloudResource interface
func (ap *CsAutoscalingPolicies) Sync() error {
	autoscalingpolicies, err := ap.cloud.Provision().AutoscalingPolicies(ap.organizationID, ap.clusterID).List()
	if err != nil {
		return err
	}

	return ap.UnmarshalToCache(autoscalingpolicies)
}

// UnmarshalToCache take the json returned from containership api
// and writes it to CsAutoscalingPolicies cache
func (ap *CsAutoscalingPolicies) UnmarshalToCache(cloudAutoscalingPolicies []types.AutoscalingPolicy) error {
	if cloudAutoscalingPolicies == nil {
		return errors.New("no autoscaling policies returned")
	}

	coerceCloudAPITypeToCerebral := make([]cerebralv1alpha1.AutoscalingPolicy, len(cloudAutoscalingPolicies))

	for i, cap := range cloudAutoscalingPolicies {
		// This converts the autoscaling policy object returned from the containership
		// API into the Cerebral AutoscalingPolicy type. This object will then get written
		// to the associated CR in camel case. In order to change to the Cerebral
		// AutoscalingPolicy type you need to reference each value to convert, since
		// the underlying types and structure is not consistent between the API representation
		// of the object and the Cerebral representation of the object.
		var scaleUp *cerebralv1alpha1.ScalingPolicyConfiguration
		var scaleDown *cerebralv1alpha1.ScalingPolicyConfiguration
		if cap.ScalingPolicy != nil {
			scaleUp = convertScalingPolicy(cap.ScalingPolicy.ScaleUp)
			scaleDown = convertScalingPolicy(cap.ScalingPolicy.ScaleDown)
		}

		ae := cerebralv1alpha1.AutoscalingPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("%s", cap.ID),
			},
			Spec: cerebralv1alpha1.AutoscalingPolicySpec{
				MetricsBackend:      cap.MetricsBackend,
				Metric:              *cap.Metric,
				MetricConfiguration: toStringMap(cap.MetricConfiguration.(map[string]interface{})),
				PollInterval:        int(*cap.PollInterval),
				SamplePeriod:        int(*cap.SamplePeriod),
				ScalingPolicy: cerebralv1alpha1.ScalingPolicy{
					ScaleUp:   scaleUp,
					ScaleDown: scaleDown,
				},
			},
		}

		coerceCloudAPITypeToCerebral[i] = ae
	}

	ap.cache = coerceCloudAPITypeToCerebral

	return nil
}

func toStringMap(config map[string]interface{}) map[string]string {
	m := make(map[string]string, len(config))
	for key, value := range config {
		m[key] = fmt.Sprintf("%s", value)
	}

	return m
}

func convertScalingPolicy(policy *types.ScalingPolicyConfiguration) *cerebralv1alpha1.ScalingPolicyConfiguration {
	// If the policy is nil, or does not contain a ComparisonOperator or AdjustmentType
	// we return nil and pretend like the policy doesn't exist and should not be written
	// to the CR
	if policy == nil ||
		policy.ComparisonOperator == nil ||
		policy.AdjustmentType == nil ||
		*policy.ComparisonOperator == "" ||
		*policy.AdjustmentType == "" {
		return nil
	}

	return &cerebralv1alpha1.ScalingPolicyConfiguration{
		Threshold:          float64(*policy.Threshold),
		ComparisonOperator: *policy.ComparisonOperator,
		AdjustmentType:     *policy.AdjustmentType,
		AdjustmentValue:    float64(*policy.AdjustmentValue),
	}
}

// Cache returns the autoscalingEngines cache
func (ap *CsAutoscalingPolicies) Cache() []cerebralv1alpha1.AutoscalingPolicy {
	return ap.cache
}

// IsEqual compares a AutoscalingEngineSpec to another AutoscalingEngine
// new, cache
func (ap *CsAutoscalingPolicies) IsEqual(specObj interface{}, parentSpecObj interface{}) (bool, error) {
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

	equal = tools.StringMapsAreEqual(spec.MetricConfiguration, autoscalingPolicy.Spec.MetricConfiguration)
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

func scalingPolicyConfigurationIsEqual(cloudPolicyConfiguration, cachePolicyConfiguration *cerebralv1alpha1.ScalingPolicyConfiguration) bool {
	if cloudPolicyConfiguration == nil && cachePolicyConfiguration == nil {
		return true
	} else if cloudPolicyConfiguration == nil || cachePolicyConfiguration == nil {
		return false
	}

	return cloudPolicyConfiguration.Threshold == cachePolicyConfiguration.Threshold &&
		cloudPolicyConfiguration.ComparisonOperator == cachePolicyConfiguration.ComparisonOperator &&
		cloudPolicyConfiguration.AdjustmentType == cachePolicyConfiguration.AdjustmentType &&
		cloudPolicyConfiguration.AdjustmentValue == cachePolicyConfiguration.AdjustmentValue
}
