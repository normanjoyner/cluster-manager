package resources

import (
	"github.com/containership/cloud-agent/internal/log"
)

// Firewalls defines the Containership Cloud Firewalls resource
type Firewalls struct {
	csResource
}

// NewFirewalls constructs a new Firewalls
func NewFirewalls() *Firewalls {
	return &Firewalls{csResource{
		Endpoint: "/organizations/{{.OrganizationID}}/clusters/{{.ClusterID}}/firewalls",
		Type:     ResourceTypeHost,
	}}
}

// GetEndpoint returns the Endpoint
func (fs *Firewalls) GetEndpoint() string {
	return fs.Endpoint
}

// GetType returns the ResourceType
func (fs *Firewalls) GetType() ResourceType {
	return fs.Type
}

// Reconcile compares created firewalls against cached firewalls
func (fs *Firewalls) Reconcile() {
	log.Info("Reconciling Firewalls...")
}

// Sync fetches firewalls from Containership Cloud and executes a callback if
// the fetched data does not match the internal cache
func (fs *Firewalls) Sync(onCacheMismatch func()) error {
	log.Info("Sync Firewalls...")
	onCacheMismatch()
	return nil
}

// Write creates firewalls on the host
func (fs *Firewalls) Write() {
	log.Info("Writing Firewalls...")
}
