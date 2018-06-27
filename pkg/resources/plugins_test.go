package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"

	containershipv3 "github.com/containership/cloud-agent/pkg/apis/containership.io/v3"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type buildIsEqualTest struct {
	inputSpec   containershipv3.PluginSpec
	inputObject *containershipv3.Plugin
	expected    bool
	message     string
}

var emptyPluginSpec = containershipv3.PluginSpec{}
var emptyPlugin = &containershipv3.Plugin{}

var plugin1spec = containershipv3.PluginSpec{
	ID:             "1",
	Description:    "description 1",
	Type:           "type",
	Version:        "1.0.0",
	Implementation: "implementation",
}

var plugin2spec = containershipv3.PluginSpec{
	ID:             "2",
	Description:    "description 2",
	Type:           "type",
	Version:        "1.0.2",
	Implementation: "implementation",
}

var plugin1 = &containershipv3.Plugin{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "plugin1",
		Namespace: "containership",
	},
	Spec: plugin1spec,
}

var plugin2 = &containershipv3.Plugin{
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
	c := NewCsPlugins()
	for _, test := range tests {
		result, err := c.IsEqual(test.inputSpec, test.inputObject)
		// Note that this is a deep compare
		assert.Equal(t, test.expected, result, test.message)
		assert.Nil(t, err)
	}
}
