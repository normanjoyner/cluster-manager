package resources

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	authv3 "github.com/containership/cluster-manager/pkg/apis/auth.containership.io/v3"
)

var emptyAuthorizationRoleSpec = authv3.AuthorizationRoleSpec{}
var emptyAuthorizationRole = &authv3.AuthorizationRole{}

var authRuleSpec1 = authv3.AuthorizationRuleSpec{
	ID:             "10",
	CreatedAt:      "123",
	UpdatedAt:      "456",
	OrganizationID: "123",
	Name:           "name",
	Description:    "description",
	OwnerID:        "456",
	Type:           authv3.AuthorizationRuleTypeKubernetes,
	APIGroups:      []string{"", "apps", "stable.example.com"},
	Resources:      []string{"pods", "nodes", "example"},
	ResourceNames:  []string{"exampleresource"},
	Verbs:          []string{"get"},
}

var authRuleSpec2 = authv3.AuthorizationRuleSpec{
	ID:             "20",
	CreatedAt:      "345",
	UpdatedAt:      "678",
	OrganizationID: "345",
	Name:           "name2",
	Description:    "description2",
	OwnerID:        "678",
	Type:           authv3.AuthorizationRuleTypeContainership,
	APIGroups:      []string{""},
	Resources:      []string{"users"},
	ResourceNames:  nil,
	Verbs:          nil,
}

var authorizationRole1 = &authv3.AuthorizationRole{
	ObjectMeta: metav1.ObjectMeta{
		Name: "name0",
	},
	Spec: authv3.AuthorizationRoleSpec{
		ID:             "1",
		CreatedAt:      "123",
		UpdatedAt:      "456",
		OrganizationID: "123",
		Name:           "name",
		Description:    "description",
		OwnerID:        "456",
		Rules:          []authv3.AuthorizationRuleSpec{authRuleSpec1, authRuleSpec2},
	},
}

func TestAuthorizationRoleIsEqual(t *testing.T) {
	c := NewCsAuthorizationRoles(nil)

	_, err := c.IsEqual("wrong type", emptyAuthorizationRole)
	assert.Error(t, err, "bad spec type")

	_, err = c.IsEqual(emptyAuthorizationRoleSpec, "wrong type")
	assert.Error(t, err, "bad parent type")

	eq, err := c.IsEqual(emptyAuthorizationRoleSpec, emptyAuthorizationRole)
	assert.NoError(t, err)
	assert.True(t, eq, "both empty")

	eq, err = c.IsEqual(emptyAuthorizationRoleSpec, authorizationRole1)
	assert.NoError(t, err)
	assert.False(t, eq, "spec only empty")

	eq, err = c.IsEqual(authorizationRole1.Spec, emptyAuthorizationRole)
	assert.NoError(t, err)
	assert.False(t, eq, "parent only empty")

	same := authorizationRole1.DeepCopy().Spec
	eq, err = c.IsEqual(same, authorizationRole1)
	assert.NoError(t, err)
	assert.True(t, eq, "copied spec")

	same = authorizationRole1.DeepCopy().Spec
	same.Rules = []authv3.AuthorizationRuleSpec{authRuleSpec2, authRuleSpec1}
	eq, err = c.IsEqual(same, authorizationRole1)
	assert.NoError(t, err)
	assert.True(t, eq, "same rules - different order doesn't matter")

	diff := authorizationRole1.DeepCopy().Spec
	diff.ID = "different"
	eq, err = c.IsEqual(diff, authorizationRole1)
	assert.NoError(t, err)
	assert.False(t, eq, "different ID")

	diff = authorizationRole1.DeepCopy().Spec
	diff.CreatedAt = "different"
	eq, err = c.IsEqual(diff, authorizationRole1)
	assert.NoError(t, err)
	assert.False(t, eq, "different created_at")

	diff = authorizationRole1.DeepCopy().Spec
	diff.UpdatedAt = "different"
	eq, err = c.IsEqual(diff, authorizationRole1)
	assert.NoError(t, err)
	assert.False(t, eq, "different updated_at")

	diff = authorizationRole1.DeepCopy().Spec
	diff.OrganizationID = "different"
	eq, err = c.IsEqual(diff, authorizationRole1)
	assert.NoError(t, err)
	assert.False(t, eq, "different organization_id")

	diff = authorizationRole1.DeepCopy().Spec
	diff.Name = "different"
	eq, err = c.IsEqual(diff, authorizationRole1)
	assert.NoError(t, err)
	assert.False(t, eq, "different name")

	diff = authorizationRole1.DeepCopy().Spec
	diff.Description = "different"
	eq, err = c.IsEqual(diff, authorizationRole1)
	assert.NoError(t, err)
	assert.False(t, eq, "different description")

	diff = authorizationRole1.DeepCopy().Spec
	diff.OwnerID = "different"
	eq, err = c.IsEqual(diff, authorizationRole1)
	assert.NoError(t, err)
	assert.False(t, eq, "different owner_id")

	diff = authorizationRole1.DeepCopy().Spec
	diff.Rules = []authv3.AuthorizationRuleSpec{}
	eq, err = c.IsEqual(diff, authorizationRole1)
	assert.NoError(t, err)
	assert.False(t, eq, "different rules - empty")

	diff = authorizationRole1.DeepCopy().Spec
	diff.Rules = []authv3.AuthorizationRuleSpec{authRuleSpec1}
	eq, err = c.IsEqual(diff, authorizationRole1)
	assert.NoError(t, err)
	assert.False(t, eq, "different rules")
}

func TestAuthorizationRolesCache(t *testing.T) {
	authorizationRoleBytes := []byte(`
[{
    "created_at": "1557429622851",
    "description": "this is a kubernetes sudo role",
    "id": "6d4203b7-aad9-45e9-8d37-d520f31cd5f0",
    "name": "kubernetes-sudo",
    "organization_id": "5813c2b6-9bf2-43ba-bc60-f3711e4e4d8a",
    "owner_id": "d5061be2-b408-47f9-8c2d-be9c14e9acaa",
    "rules": [
      {
        "api_groups": [
          "*"
        ],
        "created_at": "1557430053408",
        "description": "This rule gives full access to kubernetes",
        "id": "fab68982-3d04-4cf6-a408-cd79b2e5c9de",
        "name": "k8s-admin-rule",
        "organization_id": "5813c2b6-9bf2-43ba-bc60-f3711e4e4d8a",
        "owner_id": "d5061be2-b408-47f9-8c2d-be9c14e9acaa",
        "resource_names": [
          "*"
        ],
        "resources": [
          "*"
        ],
        "type": "kubernetes",
        "updated_at": "1557430053408",
        "verbs": [
          "get",
          "list",
          "watch",
          "create",
          "update",
          "patch",
          "delete"
        ]
      }
    ],
    "updated_at": "1557429622851"
  }]
`)

	u := NewCsAuthorizationRoles(nil)

	err := json.Unmarshal(authorizationRoleBytes, &u.cache)
	assert.NoError(t, err, "unmarshal good data")

	c := u.Cache()
	assert.Equal(t, u.cache, c)
}
