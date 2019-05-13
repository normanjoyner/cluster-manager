package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringMapsAreEqual(t *testing.T) {
	map1 := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	map2 := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	map3 := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key4": "value4",
	}

	map4 := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "differentvalueforkey",
	}

	result := StringMapsAreEqual(map1, map1)
	assert.True(t, result)
	// does not match different lengths
	result = StringMapsAreEqual(map1, map2)
	assert.False(t, result)
	// does not match same length
	result = StringMapsAreEqual(map1, map3)
	assert.False(t, result)
	// key values do not match
	result = StringMapsAreEqual(map1, map4)
	assert.False(t, result)
}

func TestStringSlicesAreEqual(t *testing.T) {
	type sliceEqualityTest struct {
		a        []string
		b        []string
		expected bool
		message  string
	}

	tests := []sliceEqualityTest{
		{
			a:        nil,
			b:        nil,
			expected: true,
			message:  "both nil",
		},
		{
			a:        nil,
			b:        []string{},
			expected: false,
			message:  "a nil, b empty",
		},
		{
			a:        []string{},
			b:        nil,
			expected: false,
			message:  "a empty, b nil",
		},
		{
			a:        []string{},
			b:        []string{},
			expected: true,
			message:  "both empty",
		},
		{
			a:        []string{"1"},
			b:        []string{},
			expected: false,
			message:  "a empty, b not empty",
		},
		{
			a:        []string{},
			b:        []string{"1"},
			expected: false,
			message:  "a not empty, b empty",
		},
		{
			a:        []string{"1", "2", "3"},
			b:        []string{"1", "3", "2"},
			expected: false,
			message:  "same elements out of order",
		},
		{
			a:        []string{"1", "2", "3"},
			b:        []string{"1", "2", "3"},
			expected: true,
			message:  "same elements in order",
		},
	}

	for _, test := range tests {
		result := StringSlicesAreEqual(test.a, test.b)
		assert.Equal(t, test.expected, result, test.message)
	}
}
