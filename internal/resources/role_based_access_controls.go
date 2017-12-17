package resources

import (
	"log"
)

type RoleBasedAccessControlsDef struct {
	csResource
}

var RoleBasedAccessControls *RoleBasedAccessControlsDef

func init() {
	RoleBasedAccessControls = &RoleBasedAccessControlsDef{csResource{"/rbac"}}
}

func (rbacs *RoleBasedAccessControlsDef) Reconcile() {
	log.Println("Reconciling rbac...")
}

func (rbacs *RoleBasedAccessControlsDef) Write() {
	log.Println("Writing rbac...")
}

func (rbacs *RoleBasedAccessControlsDef) Sync(f func()) {
	log.Println("Syncing rbac...")
	f()
}
