package constants

import (
	"testing"

	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type buildLabelTest struct {
	input    map[string]string
	expected map[string]string
}

var tests = []buildLabelTest{
	// No input
	{
		input: nil,
		expected: map[string]string{
			"containership.io/managed": "true",
		},
	},
	// One input
	{
		input: map[string]string{
			"key1": "value1",
		},
		expected: map[string]string{
			"containership.io/managed": "true",
			"key1": "value1",
		},
	},
	// Many inputs
	{
		input: map[string]string{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
			"key4": "value4",
			"key5": "value5",
		},
		expected: map[string]string{
			"containership.io/managed": "true",
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
			"key4": "value4",
			"key5": "value5",
		},
	},
}

func TestBuildContainershipLabelMap(t *testing.T) {
	for _, test := range tests {
		result := BuildContainershipLabelMap(test.input)
		// Note that this is a deep compare
		assert.Equal(t, test.expected, result)
	}
}

type buildIsContainershipTest struct {
	input    *corev1.Secret
	expected bool
	message  string
}

var isContainership = []buildIsContainershipTest{
	// No input
	{
		input:    &corev1.Secret{},
		expected: false,
		message:  "Testing empty object",
	},
	// One input not CS
	{
		input: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"key1": "value1",
				},
			},
		},
		expected: false,
		message:  "Testing object without containership managed label",
	},
	// One input
	{
		input: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"containership.io/managed": "true",
				},
			},
		},
		expected: true,
		message:  "Testing object with containership managed label",
	},
	// Many inputs
	{
		input: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"key1": "value1",
					"key2": "value2",
					"containership.io/managed": "true",
					"key4": "value4",
					"key5": "value5",
				},
			},
		},
		expected: true,
		message:  "Testing object with multiple lables, including containership managed label",
	},
}

func TestIsContainershipManaged(t *testing.T) {
	for _, test := range isContainership {
		result := IsContainershipManaged(test.input)
		// Note that this is a deep compare
		assert.Equal(t, test.expected, result, test.message)
	}
}
