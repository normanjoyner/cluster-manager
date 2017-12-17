package resources

import (
	"log"
)

// RoleBasedAccessControlsDef defines the Containership Cloud role based access controls resource
type RoleBasedAccessControlsDef struct {
	csResource
}

// RoleBasedAccessControls resource
var RoleBasedAccessControls *RoleBasedAccessControlsDef

func init() {
	RoleBasedAccessControls = &RoleBasedAccessControlsDef{csResource{"/rbac"}}
}

// Reconcile compares created role based access controls against cached role based access controls
func (rbacs *RoleBasedAccessControlsDef) Reconcile() {
	log.Println("Reconciling rbac...")
}

// Write creates role based access controls on the cluster
func (rbacs *RoleBasedAccessControlsDef) Write() {
	log.Println("Writing rbac...")
}

// Sync fetches role based access controls from Containership Cloud and executes a callback if the fetched data does not match the internal cache
func (rbacs *RoleBasedAccessControlsDef) Sync(f func()) {
	log.Println("Syncing rbac...")
	f()
}
