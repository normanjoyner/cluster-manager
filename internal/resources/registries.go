package resources

import (
	"github.com/containership/cloud-agent/internal/log"
)

// Registries defines the Containership Cloud Registries resource
type Registries struct {
	csResource
}

// NewRegistries constructs a new Registries
func NewRegistries() *Registries {
	return &Registries{csResource{
		Endpoint: "/organizations/{{.OrganizationID}}/registries",
		Type:     ResourceTypeCluster,
	}}
}

// GetEndpoint returns the Endpoint
func (rs *Registries) GetEndpoint() string {
	return rs.Endpoint
}

// GetType returns the ResourceType
func (rs *Registries) GetType() ResourceType {
	return rs.Type
}

// Reconcile compares created registries against cached registries
func (rs *Registries) Reconcile() {
	log.Info("Reconciling Registries...")
}

// Sync fetches registries from Containership Cloud and executes a callback if
// the fetched data does not match the internal cache
func (rs *Registries) Sync(onCacheMismatch func()) error {
	log.Info("Syncing Registries...")
	onCacheMismatch()
	return nil
}

// Write creates registries on the cluster
func (rs *Registries) Write() {
	log.Info("Writing Registries...")
}
