package resources

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	csv3 "github.com/containership/cluster-manager/pkg/apis/containership.io/v3"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var emptyPluginSpec = csv3.PluginSpec{}
var emptyPlugin = &csv3.Plugin{}

var plugin1 = &csv3.Plugin{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "plugin1",
		Namespace: "containership",
	},
	Spec: csv3.PluginSpec{
		ID:             "1",
		AddedAt:        "123",
		Description:    "description",
		Type:           "cni",
		Implementation: "calico",
		Version:        "2.0.0",
	},
}

func TestPluginIsEqual(t *testing.T) {
	c := NewCsPlugins(nil)

	_, err := c.IsEqual("wrong type", emptyPlugin)
	assert.Error(t, err, "bad spec type")

	_, err = c.IsEqual(emptyPluginSpec, "wrong type")
	assert.Error(t, err, "bad parent type")

	eq, err := c.IsEqual(emptyPluginSpec, emptyPlugin)
	assert.NoError(t, err)
	assert.True(t, eq, "both empty")

	eq, err = c.IsEqual(emptyPluginSpec, plugin1)
	assert.NoError(t, err)
	assert.False(t, eq, "spec only empty")

	eq, err = c.IsEqual(plugin1.Spec, emptyPlugin)
	assert.NoError(t, err)
	assert.False(t, eq, "parent only empty")

	same := plugin1.DeepCopy().Spec
	eq, err = c.IsEqual(same, plugin1)
	assert.NoError(t, err)
	assert.True(t, eq, "copied spec")

	diff := plugin1.DeepCopy().Spec
	diff.ID = "different"
	eq, err = c.IsEqual(diff, plugin1)
	assert.NoError(t, err)
	assert.False(t, eq, "different ID")

	diff = plugin1.DeepCopy().Spec
	diff.AddedAt = "different"
	eq, err = c.IsEqual(diff, plugin1)
	assert.NoError(t, err)
	assert.False(t, eq, "different added_at")

	diff = plugin1.DeepCopy().Spec
	diff.Description = "different"
	eq, err = c.IsEqual(diff, plugin1)
	assert.NoError(t, err)
	assert.False(t, eq, "different description")

	diff = plugin1.DeepCopy().Spec
	diff.Version = "different"
	eq, err = c.IsEqual(diff, plugin1)
	assert.NoError(t, err)
	assert.False(t, eq, "different version")

	diff = plugin1.DeepCopy().Spec
	diff.Implementation = "different"
	eq, err = c.IsEqual(diff, plugin1)
	assert.NoError(t, err)
	assert.False(t, eq, "different implementation")
}

func TestPluginCache(t *testing.T) {
	pluginBytes := []byte(`[{
	"id" : "1234",
	"description": "description text",
	"type": "plugin_type",
	"version": "1.0.0",
	"implementation": "plugin_implementation"
}]`)

	p := NewCsPlugins(nil)

	err := json.Unmarshal(pluginBytes, &p.cache)
	assert.NoError(t, err, "unmarshal good data")

	c := p.Cache()
	assert.Equal(t, p.cache, c)
}
