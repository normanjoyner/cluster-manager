package constants

import (
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	"github.com/containership/cluster-manager/pkg/log"
)

const (
	// ContainershipNamespace is the namespace in which all containership
	// resources will live
	ContainershipNamespace = "containership-core"
	// ContainershipServiceAccountName is the name of containership controlled
	// service account in every namespace
	ContainershipServiceAccountName = "containership"
	// KubernetesControlPlaneNamespace is the namespace that the control plane
	// components for Kubernetes are ran in
	KubernetesControlPlaneNamespace = "kube-system"
)

const (
	// ContainershipNodeIDLabelKey is the label key for the Containership node ID on nodes
	ContainershipNodeIDLabelKey = "containership.io/node-id"
	// ContainershipNodePoolIDLabelKey is the label key for the Containership node pool ID
	ContainershipNodePoolIDLabelKey = "containership.io/node-pool-id"
	// ContainershipNodeGeohashLabelKey is the label key for the Containership node geohash label
	ContainershipNodeGeohashLabelKey = "node.containership.io/geohash"
	// ContainershipClusterLabelPrefix is the prefix for cluster labels
	ContainershipClusterLabelPrefix = "cluster.containership.io/"
	// ContainershipNodePoolLabelPrefix is the prefix for node pool labels
	ContainershipNodePoolLabelPrefix = "nodepool.containership.io/"
)

const (
	// AuthorizationRoleFinalizerName is the name of the AuthorizationRole finalizer
	AuthorizationRoleFinalizerName = "authorizationrole.finalizers.containership.io"
	// AuthorizationRoleBindingFinalizerName is the name of the AuthorizationRoleBinding finalizer
	AuthorizationRoleBindingFinalizerName = "authorizationrolebinding.finalizers.containership.io"
)

const (
	// ClusterManagementPluginType is the name of the cluster management plugin type
	ClusterManagementPluginType = "cluster_management"
)

const (
	// ContainershipMount is the base Containership mount
	ContainershipMount = "/etc/containership"
)

// TODO this const block should be scoped to a sync-specific package after
// we refactor appropriately
const (
	// SyncJitterFactor is used to avoid periodic and simultaneous syncs
	SyncJitterFactor = 0.1
)

// Containership provider registry names
const (
	// EC2Registry is the name of an amazon registry in containership cloud
	EC2Registry = "amazon_ec2_registry"
	// Azure is the name of an azure registry in containership cloud
	Azure = "azure"
	// Docker is the name of a docker registry in containership cloud
	Docker = "dockerhub"
	// GCR is the name of a google cloud registry in containership cloud
	GCR = "google_registry"
	// Private is the name of a private registry in containership cloud
	Private = "private"
	// Quay is the name of a quay registry in containership cloud
	Quay = "quay"
)

// Containership managed annotations
const (
	// PluginHistoryAnnotation is used to keep track of previous versions of a plugin
	PluginHistoryAnnotation = "containership.io/plugin-history"
)

// BaseContainershipManagedLabelString is the containership
// managed label as a string
const BaseContainershipManagedLabelString = "containership.io/managed=true"

// BuildContainershipLabelMap builds a map of labels that should be attached to
// any Containership-managed resources. If additionalLabels is non-nil, they
// will be included in the returned map.
func BuildContainershipLabelMap(additionalLabels map[string]string) map[string]string {
	m := make(map[string]string)

	// Add required base label
	pair := strings.Split(BaseContainershipManagedLabelString, "=")
	m[pair[0]] = pair[1]

	for k, v := range additionalLabels {
		m[k] = v
	}

	return m
}

// IsContainershipManaged takes in a resource and looks to see if it
// is being managed by containership
func IsContainershipManaged(obj interface{}) bool {
	meta, err := meta.Accessor(obj)

	if err != nil {
		log.Error("isContainershipManaged Error: ", err)
		return false
	}

	l := meta.GetLabels()
	if cs, ok := l["containership.io/managed"]; ok && cs == "true" {
		return true
	}

	return false
}

// GetContainershipManagedSelector returns a label selector that
// selects Containership-managed resources
func GetContainershipManagedSelector() labels.Selector {
	pair := strings.Split(BaseContainershipManagedLabelString, "=")
	req, _ := labels.NewRequirement(pair[0], selection.Equals, []string{pair[1]})
	selector := labels.NewSelector()
	return selector.Add(*req)
}
