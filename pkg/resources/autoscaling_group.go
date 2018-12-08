package resources

import (
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/containership/cluster-manager/pkg/constants"
	"github.com/containership/cluster-manager/pkg/request"
	"github.com/containership/cluster-manager/pkg/tools"

	cerebralv1alpha1 "github.com/containership/cerebral/pkg/apis/cerebral.containership.io/v1alpha1"
)

// CsAutoscalingGroups defines the Containership Cloud AutoscalingGroups resource
type CsAutoscalingGroups struct {
	cloudResource
	cache []cerebralv1alpha1.AutoscalingGroup
}

// NodePool is the API object that is returned that contains the information about
// autoscaling groups
type NodePool struct {
	ID          string                  `json:"id"`
	Autoscaling APIAutoScalingGroupSpec `json:"autoscaling"`
}

// APIAutoScalingGroupSpec allows us to get the snake_case version of the
// Autoscaling Group from containership API and transform the object to be of the
// Cerebral AutoscalingGroup type
type APIAutoScalingGroupSpec struct {
	NodeSelector    map[string]string  `json:"node_selector"`
	Enabled         bool               `json:"enabled"`
	Policies        []string           `json:"policies"`
	Engine          string             `json:"engine"`
	CooldownPeriod  int                `json:"cooldown_period"`
	Suspended       bool               `json:"suspended"`
	MinNodes        int                `json:"min_nodes"`
	MaxNodes        int                `json:"max_nodes"`
	ScalingStrategy APIScalingStrategy `json:"scaling_strategy"`
}

// APIScalingStrategy is part of the APIAutoScalingGroupSpec
type APIScalingStrategy struct {
	ScaleUp   string `json:"scale_up"`
	ScaleDown string `json:"scale_down"`
}

// NewCsAutoscalingGroups constructs a new CsAutoscalingGroups
func NewCsAutoscalingGroups() *CsAutoscalingGroups {
	return &CsAutoscalingGroups{
		cloudResource: cloudResource{
			endpoint: "/organizations/{{.OrganizationID}}/clusters/{{.ClusterID}}/node-pools",
			service:  request.CloudServiceProvision,
		},
		cache: make([]cerebralv1alpha1.AutoscalingGroup, 0),
	}
}

// UnmarshalToCache take the json returned from containership API and gets the
// AutoscalingPolicy associated with them, then writes the AutoscalingGroup
// to the CsAutoscalingGroups cache
func (us *CsAutoscalingGroups) UnmarshalToCache(bytes []byte) error {
	nodepools := make([]NodePool, 0)
	err := json.Unmarshal(bytes, &nodepools)
	if err != nil {
		return err
	}

	cerebralAutoscalingGroups := make([]cerebralv1alpha1.AutoscalingGroup, 0)
	for _, np := range nodepools {
		// If autoscaling is not enabled for the node pool we should not sync the
		// node pool as an AutoscalingGroup
		if !np.Autoscaling.Enabled {
			continue
		}

		if np.Autoscaling.Engine == "" {
			np.Autoscaling.Engine = "containership"
		}

		if np.Autoscaling.NodeSelector == nil {
			np.Autoscaling.NodeSelector = map[string]string{
				constants.ContainershipNodePoolIDLabelKey: np.ID,
			}
		}

		policyIDs, err := getAutoscalingGroupPolicies(np.ID)
		if err != nil {
			return err
		}

		ag := transformAPINodePoolToCerebralASG(np, policyIDs)

		cerebralAutoscalingGroups = append(cerebralAutoscalingGroups, ag)
	}

	us.cache = cerebralAutoscalingGroups

	return nil
}

func transformAPINodePoolToCerebralASG(np NodePool, policyIDs []string) cerebralv1alpha1.AutoscalingGroup {
	return cerebralv1alpha1.AutoscalingGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name: np.ID,
		},
		Spec: cerebralv1alpha1.AutoscalingGroupSpec{
			NodeSelector:   np.Autoscaling.NodeSelector,
			Policies:       policyIDs,
			Engine:         np.Autoscaling.Engine,
			CooldownPeriod: np.Autoscaling.CooldownPeriod,
			Suspended:      np.Autoscaling.Suspended,
			MinNodes:       np.Autoscaling.MinNodes,
			MaxNodes:       np.Autoscaling.MaxNodes,
			ScalingStrategy: cerebralv1alpha1.ScalingStrategy{
				ScaleUp:   np.Autoscaling.ScalingStrategy.ScaleUp,
				ScaleDown: np.Autoscaling.ScalingStrategy.ScaleDown,
			},
		},
	}
}

// Cache return the containership AutoscalingGroup cache
func (us *CsAutoscalingGroups) Cache() []cerebralv1alpha1.AutoscalingGroup {
	return us.cache
}

// IsEqual compares a cloud AutoscalingGroupSpec to the cache AutoscalingGroup
func (us *CsAutoscalingGroups) IsEqual(specObj interface{}, parentSpecObj interface{}) (bool, error) {
	spec, ok := specObj.(cerebralv1alpha1.AutoscalingGroupSpec)
	if !ok {
		return false, fmt.Errorf("The object is not of type AutoscalingGroupSpec")
	}

	autoscalingGroup, ok := parentSpecObj.(*cerebralv1alpha1.AutoscalingGroup)
	if !ok {
		return false, fmt.Errorf("The object is not of type AutoscalingGroup")
	}

	equal := autoscalingGroup.Spec.Engine == spec.Engine &&
		autoscalingGroup.Spec.CooldownPeriod == spec.CooldownPeriod &&
		autoscalingGroup.Spec.Suspended == spec.Suspended &&
		autoscalingGroup.Spec.MinNodes == spec.MinNodes &&
		autoscalingGroup.Spec.MaxNodes == spec.MaxNodes &&
		autoscalingGroup.Spec.ScalingStrategy.ScaleUp == spec.ScalingStrategy.ScaleUp &&
		autoscalingGroup.Spec.ScalingStrategy.ScaleDown == spec.ScalingStrategy.ScaleDown

	if !equal {
		return false, nil
	}

	equal = tools.StringMapsAreEqual(spec.NodeSelector, autoscalingGroup.Spec.NodeSelector)
	if !equal {
		return false, nil
	}

	equal = policiesAreEqual(spec.Policies, autoscalingGroup.Spec.Policies)

	return equal, nil
}

func policiesAreEqual(cloudPolicies, cachePolicies []string) bool {
	if len(cloudPolicies) != len(cachePolicies) {
		return false
	}

	policiesMap := make(map[string]string, 0)
	for _, policyName := range cloudPolicies {
		policiesMap[policyName] = ""
	}

	for _, policyName := range cachePolicies {
		_, found := policiesMap[policyName]
		if !found {
			return false
		}
	}

	return true
}

func getAutoscalingGroupPolicies(nodepoolID string) ([]string, error) {
	npcr := cloudResource{
		endpoint: fmt.Sprintf("/organizations/{{.OrganizationID}}/clusters/{{.ClusterID}}/node-pools/%s/autoscaling-policies", nodepoolID),
		service:  request.CloudServiceProvision,
	}

	autoscalingpolicies, err := makeRequest(npcr.Service(), npcr.Endpoint())
	if err != nil {
		return nil, err
	}

	return getAutoscalingGroupPoliciesIDs(autoscalingpolicies)
}

func getAutoscalingGroupPoliciesIDs(autoscalingpolicies []byte) ([]string, error) {
	var aps []*CloudAPIAutoscalingPolicy
	err := json.Unmarshal(autoscalingpolicies, &aps)
	if err != nil {
		return nil, err
	}

	policiesID := make([]string, len(aps))
	for i, ap := range aps {
		policiesID[i] = ap.ID
	}

	return policiesID, nil
}
