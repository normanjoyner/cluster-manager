package constants

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
