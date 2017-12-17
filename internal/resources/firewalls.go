package resources

import (
	"log"
)

type FirewallsDef struct {
	csResource
}

var Firewalls *FirewallsDef

func init() {
	Firewalls = &FirewallsDef{csResource{"/firewalls"}}
}

func (fs *FirewallsDef) Reconcile() {
	log.Println("Reconciling Firewalls...")
}

func (fs *FirewallsDef) Write() {
	log.Println("Writing Firewalls...")
}

func (fs *FirewallsDef) Sync(f func()) {
	log.Println("Syncing Firewalls...")
}
