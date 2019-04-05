package coordinator

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	pcsv3 "github.com/containership/cluster-manager/pkg/apis/provision.containership.io/v3"
	"github.com/containership/cluster-manager/pkg/constants"
)

func TestBuildLabelMapWithExactNodePoolLabels(t *testing.T) {
	type labelTest struct {
		existingLabels map[string]string
		nodePoolLabels []*pcsv3.NodePoolLabel
		expected       map[string]string
		message        string
	}

	tests := []labelTest{
		{
			existingLabels: nil,
			nodePoolLabels: nil,
			expected:       map[string]string{},
			message:        "nil existing and nil cluster labels results in empty map",
		},
		{
			existingLabels: map[string]string{},
			nodePoolLabels: nil,
			expected:       map[string]string{},
			message:        "empty existing and nil cluster labels results in empty map",
		},
		{
			existingLabels: nil,
			nodePoolLabels: []*pcsv3.NodePoolLabel{},
			expected:       map[string]string{},
			message:        "nil existing and empty cluster labels results in empty map",
		},
		{
			existingLabels: map[string]string{},
			nodePoolLabels: []*pcsv3.NodePoolLabel{},
			expected:       map[string]string{},
			message:        "nil existing and empty cluster labels results in empty map",
		},
		{
			existingLabels: map[string]string{
				"key0": "value0",
			},
			nodePoolLabels: nil,
			expected: map[string]string{
				"key0": "value0",
			},
			message: "only non-cluster-labels existing",
		},
		{
			existingLabels: map[string]string{
				"key0":                           "value0",
				"nodepool.containership.io/key1": "value1",
				"nodepool.containership.io/key2": "",
			},
			nodePoolLabels: nil,
			expected: map[string]string{
				"key0": "value0",
			},
			message: "existing cluster label stripped",
		},
		{
			existingLabels: map[string]string{},
			nodePoolLabels: []*pcsv3.NodePoolLabel{
				{
					Spec: pcsv3.NodePoolLabelSpec{
						Key:   "nodepool.containership.io/key1",
						Value: "value1",
					},
				},
			},
			expected: map[string]string{
				"nodepool.containership.io/key1": "value1",
			},
			message: "missing cluster label added",
		},
		{
			existingLabels: map[string]string{
				"nodepool.containership.io/key1": "value1",
			},
			nodePoolLabels: []*pcsv3.NodePoolLabel{
				{
					Spec: pcsv3.NodePoolLabelSpec{
						Key:   "nodepool.containership.io/key1",
						Value: "value1",
					},
				},
			},
			expected: map[string]string{
				"nodepool.containership.io/key1": "value1",
			},
			message: "matching cluster label kept",
		},
		{
			existingLabels: map[string]string{
				"nodepool.containership.io/key1": "notmatching",
			},
			nodePoolLabels: []*pcsv3.NodePoolLabel{
				{
					Spec: pcsv3.NodePoolLabelSpec{
						Key:   "nodepool.containership.io/key1",
						Value: "value1",
					},
				},
			},
			expected: map[string]string{
				"nodepool.containership.io/key1": "value1",
			},
			message: "mismatched cluster label value",
		},
		{
			existingLabels: map[string]string{
				"nodepool.containership.io/key0": "",
				"key1":                           "value1",
				"key2":                           "",
				"nodepool.containership.io/key3": "value3",
				"nodepool.containership.io/key5": "value5",
			},
			nodePoolLabels: []*pcsv3.NodePoolLabel{
				{
					Spec: pcsv3.NodePoolLabelSpec{
						Key:   "nodepool.containership.io/key1",
						Value: "value1",
					},
				},
				{
					Spec: pcsv3.NodePoolLabelSpec{
						Key:   "nodepool.containership.io/key4",
						Value: "value4",
					},
				},
				{
					Spec: pcsv3.NodePoolLabelSpec{
						Key:   "nodepool.containership.io/key5",
						Value: "value5",
					},
				},
			},
			expected: map[string]string{
				"key1":                           "value1",
				"key2":                           "",
				"nodepool.containership.io/key1": "value1",
				"nodepool.containership.io/key4": "value4",
				"nodepool.containership.io/key5": "value5",
			},
			message: "strip unexpected and add new",
		},
	}

	for _, test := range tests {
		result := buildLabelMapWithExactNodePoolLabels(test.existingLabels, test.nodePoolLabels)
		assert.True(t, reflect.DeepEqual(result, test.expected), test.message)
	}
}

func TestNodePoolIDSelector(t *testing.T) {
	selector := nodePoolIDSelector("")
	assert.NotNil(t, selector)

	selector = nodePoolIDSelector("asdf")
	assert.NotNil(t, selector)
}

func TestIsContainershipNodePoolLabelKey(t *testing.T) {
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
			key:      constants.ContainershipNodePoolLabelPrefix,
			expected: true,
		},
		{
			key:      constants.ContainershipNodePoolLabelPrefix + "suffix",
			expected: true,
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.expected, isContainershipNodePoolLabelKey(test.key))
	}
}
