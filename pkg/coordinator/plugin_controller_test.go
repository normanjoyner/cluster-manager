package coordinator

import (
	"testing"

	"github.com/stretchr/testify/assert"

	csv3 "github.com/containership/cluster-manager/pkg/apis/containership.io/v3"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type previousVersionTest struct {
	input    *csv3.Plugin
	expected string
}

var previousVersions = []previousVersionTest{{
	input: &csv3.Plugin{
		ObjectMeta: metav1.ObjectMeta{},
	},
	expected: "",
}, {
	input: &csv3.Plugin{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"containership.io/plugin-history": "",
			},
		},
	},
	expected: "",
}, {
	input: &csv3.Plugin{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"containership.io/plugin-history": "[{\"id\":\"0a379c6c-2299-42e9-958e-148760a7a429\",\"added_at\":\"\",\"description\":\"\",\"type\":\"cni\",\"version\":\"1.0.0\",\"implementation\":\"calico\"}]",
			},
		},
	},
	expected: "1.0.0",
}, {
	input: &csv3.Plugin{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"containership.io/plugin-history": "[{\"id\":\"0a379c6c-2299-42e9-958e-148760a7a429\",\"added_at\":\"\",\"description\":\"\",\"type\":\"cni\",\"version\":\"1.0.0\",\"implementation\":\"calico\"},{\"id\":\"0a379c6c-2299-42e9-958e-148760a7a429\",\"added_at\":\"\",\"description\":\"\",\"type\":\"cni\",\"version\":\"1.0.1\",\"implementation\":\"calico\"}]",
			},
		},
	},
	expected: "1.0.1",
}}

func TestGetPreviousVersion(t *testing.T) {
	for _, test := range previousVersions {
		result := getPreviousVersion(test.input)
		assert.Equal(t, test.expected, result)
	}
}
