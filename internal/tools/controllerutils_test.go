package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"

	containershipv3 "github.com/containership/cloud-agent/pkg/apis/containership.io/v3"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	threePartKey = "kind/namespace/name"
	twoPartKey   = "kind/name"
	onePartKey   = "kind"

	kind      = "kind"
	namespace = "namespace"
	name      = "name"
)

func TestSplitMetaResourceNamespaceKeyFuncWithThreeParts(t *testing.T) {
	k, ns, n, err := SplitMetaResourceNamespaceKeyFunc(threePartKey)

	assert.Equal(t, kind, k)
	assert.Equal(t, namespace, ns)
	assert.Equal(t, name, n)
	assert.Nil(t, err)
}

func TestSplitMetaResourceNamespaceKeyFuncWithTwoParts(t *testing.T) {
	k, ns, n, err := SplitMetaResourceNamespaceKeyFunc(twoPartKey)

	assert.Equal(t, kind, k)
	// Two part keys should have empty namespace
	assert.Equal(t, "", ns)
	assert.Equal(t, name, n)
	assert.Nil(t, err)
}

func TestSplitMetaResourceNamespaceKeyFuncWithOnePart(t *testing.T) {
	_, _, _, err := SplitMetaResourceNamespaceKeyFunc(onePartKey)

	assert.NotNil(t, err)
}

func TestMetaResourceNamespaceKeyFunc(t *testing.T) {
	obj := &containershipv3.Registry{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "name",
			Namespace: "namespace",
		},
	}

	key, err := MetaResourceNamespaceKeyFunc("kind", obj)

	assert.Equal(t, threePartKey, key)
	assert.Nil(t, err)
}
