package resources

import (
	"log"
)

// RegistriesDef defines the Containership Cloud registries resource
type RegistriesDef struct {
	csResource
}

// Registries resource
var Registries *RegistriesDef

func init() {
	Registries = &RegistriesDef{csResource{"/registries"}}
}

// Reconcile compares created registries against cached registries
func (rs *RegistriesDef) Reconcile() {
	log.Println("Reconciling Registries...")
}

// Write creates registries on the cluster
func (rs *RegistriesDef) Write() {
	log.Println("Writing Registries...")
}

// Sync fetches registries from Containership Cloud and executes a callback if the fetched data does not match the internal cache
func (rs *RegistriesDef) Sync(f func()) {
	log.Println("Syncing Registries...")
	f()
}
