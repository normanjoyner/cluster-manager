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
