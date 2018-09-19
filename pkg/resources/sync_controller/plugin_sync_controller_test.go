package synccontroller

//TODO write tests for setPluginHistoryAnnotation
// test set empty
// test append
import (
	"testing"

	"github.com/stretchr/testify/assert"

	csv3 "github.com/containership/cloud-agent/pkg/apis/containership.io/v3"
	"github.com/containership/cloud-agent/pkg/constants"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type historyAnnotationTest struct {
	inputPlugin *csv3.Plugin
	expect      string
}

var setAnnotation = []historyAnnotationTest{
	{
		inputPlugin: &csv3.Plugin{
			Spec: csv3.PluginSpec{
				ID:             "id",
				Version:        "1.0.0",
				Description:    "description",
				Implementation: "implementation",
				Type:           "type",
			},
		},
		expect: "[{\"id\":\"id\",\"added_at\":\"\",\"description\":\"description\",\"type\":\"type\",\"version\":\"1.0.0\",\"implementation\":\"implementation\"}]",
	},
	{
		inputPlugin: &csv3.Plugin{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"containership.io/plugin-history": "[{\"id\":\"id\",\"added_at\":\"\",\"description\":\"description\",\"type\":\"type\",\"version\":\"1.0.0\",\"implementation\":\"implementation\"}]",
				},
			},
			Spec: csv3.PluginSpec{
				ID:             "id",
				Version:        "1.0.1",
				Description:    "description",
				Implementation: "implementation",
				Type:           "type",
			},
		},
		expect: "[{\"id\":\"id\",\"added_at\":\"\",\"description\":\"description\",\"type\":\"type\",\"version\":\"1.0.0\",\"implementation\":\"implementation\"},{\"id\":\"id\",\"added_at\":\"\",\"description\":\"description\",\"type\":\"type\",\"version\":\"1.0.1\",\"implementation\":\"implementation\"}]",
	},
	{
		inputPlugin: &csv3.Plugin{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"containership.io/plugin-history": "[{\"id\":\"id\",\"added_at\":\"\",\"description\":\"description\",\"type\":\"type\",\"version\":\"1.0.0\",\"implementation\":\"implementation\"},{\"id\":\"id\",\"added_at\":\"\",\"description\":\"description\",\"type\":\"type\",\"version\":\"1.0.1\",\"implementation\":\"implementation\"}]",
				},
			},
			Spec: csv3.PluginSpec{
				ID:             "id",
				Version:        "1.0.2",
				Description:    "description",
				Implementation: "implementation",
				Type:           "type",
			},
		},
		expect: "[{\"id\":\"id\",\"added_at\":\"\",\"description\":\"description\",\"type\":\"type\",\"version\":\"1.0.0\",\"implementation\":\"implementation\"},{\"id\":\"id\",\"added_at\":\"\",\"description\":\"description\",\"type\":\"type\",\"version\":\"1.0.1\",\"implementation\":\"implementation\"},{\"id\":\"id\",\"added_at\":\"\",\"description\":\"description\",\"type\":\"type\",\"version\":\"1.0.2\",\"implementation\":\"implementation\"}]",
	},
}

func TestSetPluginHistoryAnnotation(t *testing.T) {
	for _, test := range setAnnotation {
		pCopy := test.inputPlugin.DeepCopy()
		setPluginHistoryAnnotation(test.inputPlugin, pCopy)
		assert.Equal(t, test.expect, pCopy.Annotations[constants.PluginHistoryAnnotation])
	}

}
