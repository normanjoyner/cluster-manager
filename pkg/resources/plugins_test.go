package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"

	csv3 "github.com/containership/cluster-manager/pkg/apis/containership.io/v3"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type buildIsEqualTest struct {
	inputSpec   csv3.PluginSpec
	inputObject *csv3.Plugin
	expected    bool
	message     string
}

var emptyPluginSpec = csv3.PluginSpec{}
var emptyPlugin = &csv3.Plugin{}

var plugin1spec = csv3.PluginSpec{
	ID:             "1",
	Description:    "description 1",
	Type:           "type",
	Version:        "1.0.0",
	Implementation: "implementation",
}

var plugin2spec = csv3.PluginSpec{
	ID:             "2",
	Description:    "description 2",
	Type:           "type",
	Version:        "1.0.2",
	Implementation: "implementation",
}

var plugin1 = &csv3.Plugin{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "plugin1",
		Namespace: "containership",
	},
	Spec: plugin1spec,
}

var plugin2 = &csv3.Plugin{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "plugin2",
		Namespace: "containership",
	},
	Spec: plugin2spec,
}

var tests = []buildIsEqualTest{
	// No input
	{
		inputSpec:   emptyPluginSpec,
		inputObject: emptyPlugin,
		expected:    true,
		message:     "Both spec and object empty",
	},
	// One input
	{
		inputSpec:   emptyPluginSpec,
		inputObject: plugin1,
		expected:    false,
		message:     "Plugin spec empty with plugin object",
	},
	// One input
	{
		inputSpec:   plugin1spec,
		inputObject: emptyPlugin,
		expected:    false,
		message:     "Plugin spec with empty object",
	},
	{
		inputSpec:   plugin1spec,
		inputObject: plugin1,
		expected:    true,
		message:     "Spec plugin1 same as Object 1",
	},
	{
		inputSpec:   plugin2spec,
		inputObject: plugin2,
		expected:    true,
		message:     "Spec plugin2 same as Object 2",
	},
	{
		inputSpec:   plugin2spec,
		inputObject: plugin1,
		expected:    false,
		message:     "Spec different from Object",
	},
}

func TestPluginIsEqual(t *testing.T) {
	c := NewCsPlugins(nil)
	for _, test := range tests {
		result, err := c.IsEqual(test.inputSpec, test.inputObject)
		// Note that this is a deep compare
		assert.Equal(t, test.expected, result, test.message)
		assert.Nil(t, err)
	}

	_, err := c.IsEqual(plugin1, plugin1)
	assert.Error(t, err)

	_, err = c.IsEqual(plugin1spec, plugin1spec)
	assert.Error(t, err)
}

var pluginBytes = []byte(`[{
	"id" : "1234",
	"description": "description text",
	"type": "plugin_type",
	"version": "1.0.0",
	"implementation": "plugin_implementation"
}]`)

func TestPluginCache(t *testing.T) {
	c := NewCsPlugins(nil)
	c.cache = []csv3.PluginSpec{plugin1spec}
	ca := c.Cache()

	assert.Equal(t, c.cache, ca)
}
