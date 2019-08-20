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

func TestStringSliceContains(t *testing.T) {
	type sliceContainsTest struct {
		slice    []string
		checkFor string
		expected bool
		message  string
	}

	tests := []sliceContainsTest{
		{
			slice:    nil,
			checkFor: "",
			expected: false,
			message:  "nil slice",
		},
		{
			slice:    []string{},
			checkFor: "",
			expected: false,
			message:  "empty slice",
		},
		{
			slice:    []string{"a"},
			checkFor: "missing",
			expected: false,
			message:  "missing from non-empty slice",
		},
		{
			slice:    []string{"a"},
			checkFor: "a",
			expected: true,
			message:  "exists in single-element slice",
		},
		{
			slice:    []string{"a", "b", "c"},
			checkFor: "c",
			expected: true,
			message:  "exists at end of slice",
		},
		{
			slice:    []string{"a", "a", "a"},
			checkFor: "a",
			expected: true,
			message:  "exists with duplicates",
		},
		{
			slice:    []string{"a", "", ""},
			checkFor: "",
			expected: true,
			message:  "empty string exists",
		},
	}

	for _, test := range tests {
		result := StringSliceContains(test.slice, test.checkFor)
		assert.Equal(t, test.expected, result, test.message)
	}
}

func TestRemoveStringFromSlice(t *testing.T) {
	type removeStringFromSliceTest struct {
		slice    []string
		toRemove string
		expected []string
		message  string
	}

	tests := []removeStringFromSliceTest{
		{
			slice:    nil,
			toRemove: "",
			expected: nil,
			message:  "nil slice",
		},
		{
			slice:    []string{},
			toRemove: "",
			expected: []string{},
			message:  "empty slice",
		},
		{
			slice:    []string{"a"},
			toRemove: "a",
			expected: []string{},
			message:  "remove only element",
		},
		{
			slice:    []string{"a"},
			toRemove: "b",
			expected: []string{"a"},
			message:  "only element does not match",
		},
		{
			slice:    []string{"a", "b", "c"},
			toRemove: "a",
			expected: []string{"b", "c"},
			message:  "remove from beginning",
		},
		{
			slice:    []string{"a", "b", "c"},
			toRemove: "c",
			expected: []string{"a", "b"},
			message:  "remove from end",
		},
		{
			slice:    []string{"a", "b", "", "d", "e"},
			toRemove: "",
			expected: []string{"a", "b", "d", "e"},
			message:  "remove empty string",
		},
		{
			slice:    []string{"a", "b", "b", "b", "e"},
			toRemove: "b",
			expected: []string{"a", "e"},
			message:  "remove multiple",
		},
	}

	for _, test := range tests {
		result := RemoveStringFromSlice(test.slice, test.toRemove)
		assert.True(t, StringSlicesAreEqual(test.expected, result), test.message)
	}
}

func TestAddStringToSliceIfMissing(t *testing.T) {
	type addStringTest struct {
		slice    []string
		toAdd    string
		expected []string
		message  string
	}

	tests := []addStringTest{
		{
			slice:    nil,
			toAdd:    "",
			expected: []string{""},
			message:  "add empty string to nil slice",
		},
		{
			slice:    []string{},
			toAdd:    "",
			expected: []string{""},
			message:  "add empty string to empty slice",
		},
		{
			slice:    []string{"a", "b"},
			toAdd:    "c",
			expected: []string{"a", "b", "c"},
			message:  "add to slice",
		},
		{
			slice:    []string{"a", "b", "c"},
			toAdd:    "c",
			expected: []string{"a", "b", "c"},
			message:  "add element that already exists",
		},
		{
			slice:    []string{"c", "c", "c"},
			toAdd:    "c",
			expected: []string{"c", "c", "c"},
			message:  "add element that already exists multiple times",
		},
	}

	for _, test := range tests {
		result := AddStringToSliceIfMissing(test.slice, test.toAdd)
		assert.True(t, StringSlicesAreEqual(test.expected, result), test.message)
	}
}
