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

	map5 := map[string]string{
		"key2": "value2",
		"key1": "value1",
		"key3": "value3",
	}

	result := StringMapsAreEqual(map1, map1)
	assert.True(t, result, "compare map to itself")

	result = StringMapsAreEqual(map1, map2)
	assert.False(t, result, "different lengths")

	result = StringMapsAreEqual(map1, map3)
	assert.False(t, result, "same length, different contents")

	result = StringMapsAreEqual(map1, map4)
	assert.False(t, result, "same keys, different value")

	result = StringMapsAreEqual(map1, map5)
	assert.True(t, result, "same contents, different order")
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
