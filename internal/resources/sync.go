package resources

// ResourceType defines the type of a resource
type ResourceType int

const (
	// ResourceTypeHost is a resource that lives at the host level
	ResourceTypeHost = iota
	// ResourceTypeCluster is a resource that lives at the cluster level
	ResourceTypeCluster
)

// Syncable defines an interface for resources to adhere to in order to be kept
// in sync with Containership Cloud
type Syncable interface {
	// GetEndpoint returns the API endpoint associated with this syncable
	GetEndpoint() string
	// GetType returns the type of this resource
	GetType() ResourceType
	// Reconcile checks actual resources on server vs cached
	Reconcile()
	// Sync fetches resource from Cloud API, compares against cache
	// takes in function that is executed if states do not match
	Sync(onCacheMismatch func()) error
	// Write updates state to match desired state
	Write()
}

// CSResource defines what each construct needs to contain
type csResource struct {
	Endpoint string
	Type     ResourceType
}
