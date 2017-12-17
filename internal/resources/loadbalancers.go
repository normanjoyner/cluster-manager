package resources

import (
	"log"
)

// LoadbalancersDef defines the Containership Cloud loadbalancer resource
type LoadbalancersDef struct {
	csResource
}

// Loadbalancers resource
var Loadbalancers *LoadbalancersDef

func init() {
	Loadbalancers = &LoadbalancersDef{csResource{"/loadbalancers"}}
}

// Reconcile compares created loadbalancers against cached loadbalancers
func (lbs *LoadbalancersDef) Reconcile() {
	log.Println("Reconciling Loadbalancers...")
}

// Write creates loadbalancers on the cluster
func (lbs *LoadbalancersDef) Write() {
	log.Println("Writing Loadbalancers...")
}

// Sync fetches loadbalancers from Containership Cloud and executes a callback if the fetched data does not match the internal cache
func (lbs *LoadbalancersDef) Sync(f func()) {
	log.Println("Syncing Loadbalancers...")
	f()
}
