package resources

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	csv3 "github.com/containership/cluster-manager/pkg/apis/containership.io/v3"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var emptyRegistrySpec = csv3.RegistrySpec{}
var emptyRegistry = &csv3.Registry{}

var registry1 = &csv3.Registry{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "registry1",
		Namespace: "containership",
	},
	Spec: csv3.RegistrySpec{
		ID:            "1",
		Description:   "description 1",
		Organization:  "1234-234-567",
		Email:         "",
		Serveraddress: "hub.docker.com",
		Provider:      "amazon_ec2_registry",
		Credentials: map[string]string{
			"key": "value",
		},
		Owner: "",
	},
}

func TestRegistryIsEqual(t *testing.T) {
	c := NewCsRegistries(nil)

	_, err := c.IsEqual("wrong type", emptyRegistry)
	assert.Error(t, err, "bad spec type")

	_, err = c.IsEqual(emptyRegistrySpec, "wrong type")
	assert.Error(t, err, "bad parent type")

	eq, err := c.IsEqual(emptyRegistrySpec, emptyRegistry)
	assert.NoError(t, err)
	assert.True(t, eq, "both empty")

	eq, err = c.IsEqual(emptyRegistrySpec, registry1)
	assert.NoError(t, err)
	assert.False(t, eq, "spec only empty")

	eq, err = c.IsEqual(registry1.Spec, emptyRegistry)
	assert.NoError(t, err)
	assert.False(t, eq, "parent only empty")

	same := registry1.DeepCopy().Spec
	eq, err = c.IsEqual(same, registry1)
	assert.NoError(t, err)
	assert.True(t, eq, "copied spec")

	diff := registry1.DeepCopy().Spec
	diff.ID = "different"
	eq, err = c.IsEqual(diff, registry1)
	assert.NoError(t, err)
	assert.False(t, eq, "different ID")

	diff = registry1.DeepCopy().Spec
	diff.AddedAt = "different"
	eq, err = c.IsEqual(diff, registry1)
	assert.NoError(t, err)
	assert.False(t, eq, "different added_at")

	diff = registry1.DeepCopy().Spec
	diff.Description = "different"
	eq, err = c.IsEqual(diff, registry1)
	assert.NoError(t, err)
	assert.False(t, eq, "different description")

	diff = registry1.DeepCopy().Spec
	diff.Organization = "different"
	eq, err = c.IsEqual(diff, registry1)
	assert.NoError(t, err)
	assert.False(t, eq, "different org")

	diff = registry1.DeepCopy().Spec
	diff.Email = "different"
	eq, err = c.IsEqual(diff, registry1)
	assert.NoError(t, err)
	assert.False(t, eq, "different email")

	diff = registry1.DeepCopy().Spec
	diff.Serveraddress = "different"
	eq, err = c.IsEqual(diff, registry1)
	assert.NoError(t, err)
	assert.False(t, eq, "different server address")

	diff = registry1.DeepCopy().Spec
	diff.Provider = "different"
	eq, err = c.IsEqual(diff, registry1)
	assert.NoError(t, err)
	assert.False(t, eq, "different provider")

	diff = registry1.DeepCopy().Spec
	diff.Owner = "different"
	eq, err = c.IsEqual(diff, registry1)
	assert.NoError(t, err)
	assert.False(t, eq, "different owner")

	diff = registry1.DeepCopy().Spec
	diff.Credentials = nil
	eq, err = c.IsEqual(diff, registry1)
	assert.NoError(t, err)
	assert.False(t, eq, "different credentials - one nil")

	diff = registry1.DeepCopy().Spec
	diff.Credentials = map[string]string{}
	eq, err = c.IsEqual(diff, registry1)
	assert.NoError(t, err)
	assert.False(t, eq, "different credentials - one empty")

	diff = registry1.DeepCopy().Spec
	diff.Credentials["key"] = "different"
	eq, err = c.IsEqual(diff, registry1)
	assert.NoError(t, err)
	assert.False(t, eq, "different credentials - different value")
}

func TestRegistriesCache(t *testing.T) {
	registryBytes := []byte(`[{
	"id": "1234",
	"added_at": "addedtimestamp",
	"description": "description",
	"organization_id": "organization-uuid",
	"email": "testing@email.com",
	"serveraddress": "https://docker.com",
	"provider": "docker",
	"credentials": {
		"key": "value"
	},
	"owner": "uuid",
	"authToken": {
		"token": "token",
		"endpoint": "/something",
		"type": "type",
		"expires": "datetime"
	}
}]`)

	r := NewCsRegistries(nil)

	err := json.Unmarshal(registryBytes, &r.cache)
	assert.NoError(t, err, "unmarshal good data")

	c := r.Cache()
	assert.Equal(t, r.cache, c)
}
