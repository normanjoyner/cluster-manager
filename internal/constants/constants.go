package constants

const (
	// ContainershipNamespace is that namespace in which all containership
	// resources will live
	ContainershipNamespace = "containership-core"
	// ContainershipServiceAccountName is the name of containership controlled
	// service account in every namespace
	ContainershipServiceAccountName = "containership"
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

// BaseContainershipManagedLabel is a label for containership type resources
// easy filtering
// Unfortunately there's no const maps in go...please don't modify this.
var BaseContainershipManagedLabel = map[string]string{
	"containershio.io": "managed",
}

// BaseContainershipManagedLabelString is the containership
// managed label as a string
// TODO this is gross
const BaseContainershipManagedLabelString = "containership.io=managed"
