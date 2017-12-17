package resources

import (
	"log"
)

type RegistriesDef struct {
	csResource
}

var Registries *RegistriesDef

func init() {
	Registries = &RegistriesDef{csResource{"/registries"}}
}

func (rs *RegistriesDef) Reconcile() {
	log.Println("Reconciling Registries...")
}

func (rs *RegistriesDef) Write() {
	log.Println("Writing Registries...")
}

func (rs *RegistriesDef) Sync(f func()) {
	log.Println("Syncing Registries...")
	f()
}
