package tools

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
)

// MetaResourceNamespaceKeyFunc is a convenient KeyFunc which knows how to make
// keys for API objects which implement meta.Interface.
// The key uses the format <kind>/<namespace>/<name> unless <namespace> is empty,
// then it's <kind>/<name>.
func MetaResourceNamespaceKeyFunc(kind string, obj interface{}) (string, error) {
	if key, ok := obj.(string); ok {
		return string(key), nil
	}
	m, err := meta.Accessor(obj)

	if err != nil {
		return "", fmt.Errorf("object has no meta: %v", err)
	}

	if len(m.GetNamespace()) > 0 {
		return kind + "/" + m.GetNamespace() + "/" + m.GetName(), nil
	}
	return kind + "/" + m.GetName(), nil
}

// SplitMetaResourceNamespaceKeyFunc returns the kind, namespace and name that
// MetaResourceNamespaceKeyFunc encoded into key.
func SplitMetaResourceNamespaceKeyFunc(key string) (kind, namespace, name string, err error) {
	parts := strings.Split(key, "/")
	switch len(parts) {
	case 2:
		// kind and name only
		return parts[0], "", parts[1], nil
	case 3:
		// kind, namespace and name
		return parts[0], parts[1], parts[2], nil
	}

	return "", "", "", fmt.Errorf("unexpected key format: %q", key)
}
