package resources

import (
	"encoding/json"

	"github.com/pkg/errors"

	cscloud "github.com/containership/csctl/cloud"

	authv3 "github.com/containership/cluster-manager/pkg/apis/auth.containership.io/v3"
	"github.com/containership/cluster-manager/pkg/tools"
)

// CsAuthorizationRoles defines the Containership Cloud AuthorizationRoles resource
type CsAuthorizationRoles struct {
	cloudResource
	cache []authv3.AuthorizationRoleSpec
}

// NewCsAuthorizationRoles constructs a new CsAuthorizationRoles
func NewCsAuthorizationRoles(cloud cscloud.Interface) *CsAuthorizationRoles {
	cache := make([]authv3.AuthorizationRoleSpec, 0)
	return &CsAuthorizationRoles{
		newCloudResource(cloud),
		cache,
	}
}

// Sync implements the CloudResource interface
func (l *CsAuthorizationRoles) Sync() error {
	authorizationRoles, err := l.cloud.Auth().AuthorizationRoles(l.organizationID).KubernetesClusterRBAC(l.clusterID)
	if err != nil {
		return errors.Wrap(err, "listing authorization roles")
	}

	data, err := json.Marshal(authorizationRoles)
	if err != nil {
		return errors.Wrap(err, "marshaling authorization roles")
	}

	json.Unmarshal(data, &l.cache)

	return nil
}

// Cache return the containership authorizationRoles cache
func (l *CsAuthorizationRoles) Cache() []authv3.AuthorizationRoleSpec {
	return l.cache
}

// IsEqual compares a AuthorizationRoleSpec to another AuthorizationRole
func (l *CsAuthorizationRoles) IsEqual(specObj interface{}, parentSpecObj interface{}) (bool, error) {
	spec, ok := specObj.(authv3.AuthorizationRoleSpec)
	if !ok {
		return false, errors.New("object is not of type AuthorizationRoleSpec")
	}

	authorizationRole, ok := parentSpecObj.(*authv3.AuthorizationRole)
	if !ok {
		return false, errors.New("object is not of type AuthorizationRole")
	}

	equal := spec.ID == authorizationRole.Spec.ID &&
		spec.CreatedAt == authorizationRole.Spec.CreatedAt &&
		spec.UpdatedAt == authorizationRole.Spec.UpdatedAt &&
		spec.OrganizationID == authorizationRole.Spec.OrganizationID &&
		spec.Name == authorizationRole.Spec.Name &&
		spec.Description == authorizationRole.Spec.Description &&
		spec.OwnerID == authorizationRole.Spec.OwnerID

	if !equal {
		return false, nil
	}

	byID := make(map[string]authv3.AuthorizationRuleSpec, 0)
	for _, rule := range authorizationRole.Spec.Rules {
		byID[rule.ID] = rule
	}

	return authorizationRulesEqualCompare(spec.Rules, byID), nil
}

func authorizationRulesAreEqual(a authv3.AuthorizationRuleSpec, b authv3.AuthorizationRuleSpec) bool {
	return a.ID == b.ID &&
		a.Name == b.Name &&
		a.CreatedAt == b.CreatedAt &&
		a.UpdatedAt == b.UpdatedAt &&
		a.OrganizationID == b.OrganizationID &&
		a.Description == b.Description &&
		a.OwnerID == b.OwnerID &&
		a.Type == b.Type &&
		tools.StringSlicesAreEqual(a.APIGroups, b.APIGroups) &&
		tools.StringSlicesAreEqual(a.Resources, b.Resources) &&
		tools.StringSlicesAreEqual(a.ResourceNames, b.ResourceNames) &&
		tools.StringSlicesAreEqual(a.Verbs, b.Verbs)
}

func authorizationRulesEqualCompare(specAuthorizationRules []authv3.AuthorizationRuleSpec, authorizationRulesByID map[string]authv3.AuthorizationRuleSpec) bool {
	if len(specAuthorizationRules) != len(authorizationRulesByID) {
		return false
	}

	specAuthorizationRulesByID := make(map[string]authv3.AuthorizationRuleSpec, 0)
	for _, rule := range specAuthorizationRules {
		specAuthorizationRulesByID[rule.ID] = rule
	}

	for id, rule := range authorizationRulesByID {
		if !authorizationRulesAreEqual(rule, specAuthorizationRulesByID[id]) {
			return false
		}
	}

	return true
}
