package resources

import (
	"log"
)

// FirewallsDef defines the Containership Cloud firewall resource
type FirewallsDef struct {
	csResource
}

// Firewalls resource
var Firewalls *FirewallsDef

func init() {
	Firewalls = &FirewallsDef{csResource{"/firewalls"}}
}

// Reconcile compares created firewalls against cached firewalls
func (fs *FirewallsDef) Reconcile() {
	log.Println("Reconciling Firewalls...")
}

// Write creates firewalls on the host
func (fs *FirewallsDef) Write() {
	log.Println("Writing Firewalls...")
}

// Sync fetches firewalls from Containership Cloud and executes a callback if the fetched data does not match the internal cache
func (fs *FirewallsDef) Sync(f func()) {
	log.Println("Syncing Firewalls...")
	f()
}
