package resources

import (
	"encoding/json"

	"github.com/pkg/errors"

	cscloud "github.com/containership/csctl/cloud"

	authv3 "github.com/containership/cluster-manager/pkg/apis/auth.containership.io/v3"
)

// CsAuthorizationRoleBindings defines the Containership Cloud AuthorizationRoleBindings resource
type CsAuthorizationRoleBindings struct {
	cloudResource
	cache []authv3.AuthorizationRoleBindingSpec
}

// NewCsAuthorizationRoleBindings constructs a new CsAuthorizationRoleBindings
func NewCsAuthorizationRoleBindings(cloud cscloud.Interface) *CsAuthorizationRoleBindings {
	cache := make([]authv3.AuthorizationRoleBindingSpec, 0)
	return &CsAuthorizationRoleBindings{
		newCloudResource(cloud),
		cache,
	}
}

// Sync implements the CloudResource interface
func (l *CsAuthorizationRoleBindings) Sync() error {
	authorizationRoleBindings, err := l.cloud.Auth().AuthorizationRoleBindings(l.organizationID).ListForCluster(l.clusterID)
	if err != nil {
		return errors.Wrap(err, "listing authorization role bindings")
	}

	data, err := json.Marshal(authorizationRoleBindings)
	if err != nil {
		return errors.Wrap(err, "marshaling authorization role bindings")
	}

	json.Unmarshal(data, &l.cache)

	return nil
}

// Cache return the containership authorizationRoleBindings cache
func (l *CsAuthorizationRoleBindings) Cache() []authv3.AuthorizationRoleBindingSpec {
	return l.cache
}

// IsEqual compares a AuthorizationRoleBindingSpec to another AuthorizationRoleBinding
func (l *CsAuthorizationRoleBindings) IsEqual(specObj interface{}, parentSpecObj interface{}) (bool, error) {
	spec, ok := specObj.(authv3.AuthorizationRoleBindingSpec)
	if !ok {
		return false, errors.New("object is not of type AuthorizationRoleBindingSpec")
	}

	authorizationRoleBinding, ok := parentSpecObj.(*authv3.AuthorizationRoleBinding)
	if !ok {
		return false, errors.New("object is not of type AuthorizationRoleBinding")
	}

	equal := spec.ID == authorizationRoleBinding.Spec.ID &&
		spec.CreatedAt == authorizationRoleBinding.Spec.CreatedAt &&
		spec.UpdatedAt == authorizationRoleBinding.Spec.UpdatedAt &&
		spec.OrganizationID == authorizationRoleBinding.Spec.OrganizationID &&
		spec.OwnerID == authorizationRoleBinding.Spec.OwnerID &&
		spec.Type == authorizationRoleBinding.Spec.Type &&
		spec.AuthorizationRoleID == authorizationRoleBinding.Spec.AuthorizationRoleID &&
		spec.UserID == authorizationRoleBinding.Spec.UserID &&
		spec.TeamID == authorizationRoleBinding.Spec.TeamID &&
		spec.ClusterID == authorizationRoleBinding.Spec.ClusterID &&
		spec.Namespace == authorizationRoleBinding.Spec.Namespace

	return equal, nil
}
