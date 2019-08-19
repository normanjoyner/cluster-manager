package tools

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	"github.com/containership/cluster-manager/pkg/log"
)

const (
	// IndexByIDFunctionName is the name of the index function we define to
	// access items by ID.
	IndexByIDFunctionName = "byID"
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

// IndexByIDKeyFun returns a function for indexing a cache by ID
func IndexByIDKeyFun() cache.Indexers {
	return cache.Indexers{
		IndexByIDFunctionName: func(obj interface{}) ([]string, error) {
			meta, err := meta.Accessor(obj)
			if err != nil {
				return []string{""}, fmt.Errorf("object has no meta: %v", err)
			}
			return []string{meta.GetName()}, nil
		},
	}
}

// CreateAndStartRecorder creates and starts a new recorder with the given name
func CreateAndStartRecorder(kubeclientset kubernetes.Interface, name string) record.EventRecorder {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(log.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{
		Interface: kubeclientset.CoreV1().Events(""),
	})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{
		Component: name,
	})

	return recorder
}
