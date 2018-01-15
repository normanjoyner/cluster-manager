package constants

import (
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"

	"github.com/containership/cloud-agent/internal/log"
)

const (
	// ContainershipNamespace is that namespace in which all containership
	// resources will live
	ContainershipNamespace = "containership-core"
	// ContainershipServiceAccountName is the name of containership controlled
	// service account in every namespace
	ContainershipServiceAccountName = "containership"
)

const (
	// ContainershipMount is the base Containership mount
	ContainershipMount = "/etc/containership"
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
