package coordinator

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	csv3 "github.com/containership/cluster-manager/pkg/apis/containership.io/v3"
	"github.com/containership/cluster-manager/pkg/constants"
)

func TestBuildLabelMapWithExactClusterLabels(t *testing.T) {
	type labelTest struct {
		existingLabels map[string]string
		clusterLabels  []*csv3.ClusterLabel
		expected       map[string]string
		message        string
	}

	tests := []labelTest{
		{
			existingLabels: nil,
			clusterLabels:  nil,
			expected:       map[string]string{},
			message:        "nil existing and nil cluster labels results in empty map",
		},
		{
			existingLabels: map[string]string{},
			clusterLabels:  nil,
			expected:       map[string]string{},
			message:        "empty existing and nil cluster labels results in empty map",
		},
		{
			existingLabels: nil,
			clusterLabels:  []*csv3.ClusterLabel{},
			expected:       map[string]string{},
			message:        "nil existing and empty cluster labels results in empty map",
		},
		{
			existingLabels: map[string]string{},
			clusterLabels:  []*csv3.ClusterLabel{},
			expected:       map[string]string{},
			message:        "nil existing and empty cluster labels results in empty map",
		},
		{
			existingLabels: map[string]string{
				"key0": "value0",
			},
			clusterLabels: nil,
			expected: map[string]string{
				"key0": "value0",
			},
			message: "only non-cluster-labels existing",
		},
		{
			existingLabels: map[string]string{
				"key0":                          "value0",
				"cluster.containership.io/key1": "value1",
				"cluster.containership.io/key2": "",
			},
			clusterLabels: nil,
			expected: map[string]string{
				"key0": "value0",
			},
			message: "existing cluster label stripped",
		},
		{
			existingLabels: map[string]string{},
			clusterLabels: []*csv3.ClusterLabel{
				{
					Spec: csv3.ClusterLabelSpec{
						Key:   "cluster.containership.io/key1",
						Value: "value1",
					},
				},
			},
			expected: map[string]string{
				"cluster.containership.io/key1": "value1",
			},
			message: "missing cluster label added",
		},
		{
			existingLabels: map[string]string{
				"cluster.containership.io/key1": "value1",
			},
			clusterLabels: []*csv3.ClusterLabel{
				{
					Spec: csv3.ClusterLabelSpec{
						Key:   "cluster.containership.io/key1",
						Value: "value1",
					},
				},
			},
			expected: map[string]string{
				"cluster.containership.io/key1": "value1",
			},
			message: "matching cluster label kept",
		},
		{
			existingLabels: map[string]string{
				"cluster.containership.io/key1": "notmatching",
			},
			clusterLabels: []*csv3.ClusterLabel{
				{
					Spec: csv3.ClusterLabelSpec{
						Key:   "cluster.containership.io/key1",
						Value: "value1",
					},
				},
			},
			expected: map[string]string{
				"cluster.containership.io/key1": "value1",
			},
			message: "mismatched cluster label value",
		},
		{
			existingLabels: map[string]string{
				"cluster.containership.io/key0": "",
				"key1":                          "value1",
				"key2":                          "",
				"cluster.containership.io/key3": "value3",
				"cluster.containership.io/key5": "value5",
			},
			clusterLabels: []*csv3.ClusterLabel{
				{
					Spec: csv3.ClusterLabelSpec{
						Key:   "cluster.containership.io/key1",
						Value: "value1",
					},
				},
				{
					Spec: csv3.ClusterLabelSpec{
						Key:   "cluster.containership.io/key4",
						Value: "value4",
					},
				},
				{
					Spec: csv3.ClusterLabelSpec{
						Key:   "cluster.containership.io/key5",
						Value: "value5",
					},
				},
			},
			expected: map[string]string{
				"key1":                          "value1",
				"key2":                          "",
				"cluster.containership.io/key1": "value1",
				"cluster.containership.io/key4": "value4",
				"cluster.containership.io/key5": "value5",
			},
			message: "strip unexpected and add new",
		},
	}

	for _, test := range tests {
		result := buildLabelMapWithExactClusterLabels(test.existingLabels, test.clusterLabels)
		assert.True(t, reflect.DeepEqual(result, test.expected), test.message)
	}
}

func TestIsContainershipClusterLabelKey(t *testing.T) {
	type keyTest struct {
		key      string
		expected bool
	}

	tests := []keyTest{
		{
			key:      "",
			expected: false,
		},
		{
			key:      "nope",
			expected: false,
		},
		{
			key:      constants.ContainershipClusterLabelPrefix,
			expected: true,
		},
		{
			key:      constants.ContainershipClusterLabelPrefix + "suffix",
			expected: true,
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.expected, isContainershipClusterLabelKey(test.key))
	}
}
