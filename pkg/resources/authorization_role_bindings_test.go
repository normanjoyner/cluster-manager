package resources

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	authv3 "github.com/containership/cluster-manager/pkg/apis/auth.containership.io/v3"
)

var emptyAuthorizationRoleBindingSpec = authv3.AuthorizationRoleBindingSpec{}
var emptyAuthorizationRoleBinding = &authv3.AuthorizationRoleBinding{}

var authorizationRoleBinding1 = &authv3.AuthorizationRoleBinding{
	ObjectMeta: metav1.ObjectMeta{
		Name: "name0",
	},
	Spec: authv3.AuthorizationRoleBindingSpec{
		ID:                  "1",
		CreatedAt:           "123",
		UpdatedAt:           "456",
		OrganizationID:      "123",
		OwnerID:             "456",
		Type:                authv3.AuthorizationRoleBindingTypeUser,
		AuthorizationRoleID: "789",
		UserID:              "12345",
	},
}

func TestAuthorizationRoleBindingIsEqual(t *testing.T) {
	c := NewCsAuthorizationRoleBindings(nil)

	_, err := c.IsEqual("wrong type", emptyAuthorizationRoleBinding)
	assert.Error(t, err, "bad spec type")

	_, err = c.IsEqual(emptyAuthorizationRoleBindingSpec, "wrong type")
	assert.Error(t, err, "bad parent type")

	eq, err := c.IsEqual(emptyAuthorizationRoleBindingSpec, emptyAuthorizationRoleBinding)
	assert.NoError(t, err)
	assert.True(t, eq, "both empty")

	eq, err = c.IsEqual(emptyAuthorizationRoleBindingSpec, authorizationRoleBinding1)
	assert.NoError(t, err)
	assert.False(t, eq, "spec only empty")

	eq, err = c.IsEqual(authorizationRoleBinding1.Spec, emptyAuthorizationRoleBinding)
	assert.NoError(t, err)
	assert.False(t, eq, "parent only empty")

	same := authorizationRoleBinding1.DeepCopy().Spec
	eq, err = c.IsEqual(same, authorizationRoleBinding1)
	assert.NoError(t, err)
	assert.True(t, eq, "copied spec")

	diff := authorizationRoleBinding1.DeepCopy().Spec
	diff.ID = "different"
	eq, err = c.IsEqual(diff, authorizationRoleBinding1)
	assert.NoError(t, err)
	assert.False(t, eq, "different ID")

	diff = authorizationRoleBinding1.DeepCopy().Spec
	diff.CreatedAt = "different"
	eq, err = c.IsEqual(diff, authorizationRoleBinding1)
	assert.NoError(t, err)
	assert.False(t, eq, "different created_at")

	diff = authorizationRoleBinding1.DeepCopy().Spec
	diff.UpdatedAt = "different"
	eq, err = c.IsEqual(diff, authorizationRoleBinding1)
	assert.NoError(t, err)
	assert.False(t, eq, "different updated_at")

	diff = authorizationRoleBinding1.DeepCopy().Spec
	diff.OrganizationID = "different"
	eq, err = c.IsEqual(diff, authorizationRoleBinding1)
	assert.NoError(t, err)
	assert.False(t, eq, "different organization_id")

	diff = authorizationRoleBinding1.DeepCopy().Spec
	diff.OwnerID = "different"
	eq, err = c.IsEqual(diff, authorizationRoleBinding1)
	assert.NoError(t, err)
	assert.False(t, eq, "different owner_id")

	diff = authorizationRoleBinding1.DeepCopy().Spec
	diff.Type = authv3.AuthorizationRoleBindingTypeTeam
	eq, err = c.IsEqual(diff, authorizationRoleBinding1)
	assert.NoError(t, err)
	assert.False(t, eq, "different type")

	diff = authorizationRoleBinding1.DeepCopy().Spec
	diff.AuthorizationRoleID = "different"
	eq, err = c.IsEqual(diff, authorizationRoleBinding1)
	assert.NoError(t, err)
	assert.False(t, eq, "different auth role ID")

	diff = authorizationRoleBinding1.DeepCopy().Spec
	diff.UserID = "different"
	eq, err = c.IsEqual(diff, authorizationRoleBinding1)
	assert.NoError(t, err)
	assert.False(t, eq, "different user ID")

	diff = authorizationRoleBinding1.DeepCopy().Spec
	diff.TeamID = "different"
	eq, err = c.IsEqual(diff, authorizationRoleBinding1)
	assert.NoError(t, err)
	assert.False(t, eq, "different team ID")

	diff = authorizationRoleBinding1.DeepCopy().Spec
	diff.ClusterID = "different"
	eq, err = c.IsEqual(diff, authorizationRoleBinding1)
	assert.NoError(t, err)
	assert.False(t, eq, "different cluster ID")

	diff = authorizationRoleBinding1.DeepCopy().Spec
	diff.Namespace = "different"
	eq, err = c.IsEqual(diff, authorizationRoleBinding1)
	assert.NoError(t, err)
	assert.False(t, eq, "different namespace")
}

func TestAuthorizationRoleBindingsCache(t *testing.T) {
	authorizationRoleBindingBytes := []byte(`
[{
  "authorization_role_id": "f814bc1e-51d2-4ddb-8db6-ff24a229f9cb",
  "created_at": "1557495552344",
  "id": "ce64bd93-259d-4c6d-9038-fbf30e19b545",
  "organization_id": "5813c2b6-9bf2-43ba-bc60-f3711e4e4d8a",
  "owner_id": "d5061be2-b408-47f9-8c2d-be9c14e9acaa",
  "type": "UserBinding",
  "updated_at": "1557495552344",
  "user_id": "d5061be2-b408-47f9-8c2d-be9c14e9acaa"
}]
`)

	u := NewCsAuthorizationRoleBindings(nil)

	err := json.Unmarshal(authorizationRoleBindingBytes, &u.cache)
	assert.NoError(t, err, "unmarshal good data")

	c := u.Cache()
	assert.Equal(t, u.cache, c)
}
