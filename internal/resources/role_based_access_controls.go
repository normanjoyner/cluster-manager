package resources

import (
	"github.com/containership/cloud-agent/internal/log"
)

// RoleBasedAccessControls defines the Containership Cloud role based access controls resource
type RoleBasedAccessControls struct {
	csResource
}

// NewRoleBasedAccessControls constructs a new RoleBasedAccessControls
func NewRoleBasedAccessControls() *RoleBasedAccessControls {
	return &RoleBasedAccessControls{csResource{
		Endpoint: "/organizations/{{.OrganizationID}}/rbac",
		Type:     ResourceTypeCluster,
	}}
}

// GetEndpoint returns the Endpoint
func (rbacs *RoleBasedAccessControls) GetEndpoint() string {
	return rbacs.Endpoint
}

// GetType returns the ResourceType
func (rbacs *RoleBasedAccessControls) GetType() ResourceType {
	return rbacs.Type
}

// Reconcile compares created role based access controls against cached role based access controls
func (rbacs *RoleBasedAccessControls) Reconcile() {
	log.Info("Reconciling rbac...")
}

// Sync fetches role based access controls from Containership Cloud and
// executes a callback if the fetched data does not match the internal cache
func (rbacs *RoleBasedAccessControls) Sync(onCacheMismatch func()) error {
	log.Info("Syncing rbac...")
	onCacheMismatch()
	return nil
}

// Write creates role based access controls on the cluster
func (rbacs *RoleBasedAccessControls) Write() {
	log.Info("Writing rbac...")
}
