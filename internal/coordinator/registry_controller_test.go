package coordinator

import (
	"testing"

	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
)

var longLocalObjectReference = []corev1.LocalObjectReference{
	{Name: "value1"},
	{Name: "value2"},
	{Name: "value3"},
	{Name: "value4"},
	{Name: "value5"},
}

var longLocalObjectReferenceSame = []corev1.LocalObjectReference{
	{Name: "value1"},
	{Name: "value2"},
	{Name: "value3"},
	{Name: "value4"},
	{Name: "value5"},
}

var shortLocalObjectReference = []corev1.LocalObjectReference{
	{Name: "value1"},
	{Name: "value2"},
}

var shortLocalObjectReferenceDiff = []corev1.LocalObjectReference{
	{Name: "value1"},
	{Name: "value3"},
}

var empty = make([]corev1.LocalObjectReference, 0)

func TestAreImagePullSecretsEqual(t *testing.T) {
	same := areImagePullSecretsEqual(longLocalObjectReference, longLocalObjectReferenceSame)
	assert.Equal(t, true, same)

	diff := areImagePullSecretsEqual(longLocalObjectReference, shortLocalObjectReference)
	assert.Equal(t, false, diff)

	diffSameSize := areImagePullSecretsEqual(shortLocalObjectReferenceDiff, shortLocalObjectReference)
	assert.Equal(t, false, diffSameSize)

	emptySame := areImagePullSecretsEqual(empty, empty)
	assert.Equal(t, true, emptySame)

	emptyDiff := areImagePullSecretsEqual(empty, shortLocalObjectReferenceDiff)
	assert.Equal(t, false, emptyDiff)

	emptyDiffOther := areImagePullSecretsEqual(shortLocalObjectReferenceDiff, empty)
	assert.Equal(t, false, emptyDiffOther)
}
