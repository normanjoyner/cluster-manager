package resources

import (
	"log"
)

type LoadbalancersDef struct {
	csResource
}

var Loadbalancers *LoadbalancersDef

func init() {
	Loadbalancers = &LoadbalancersDef{csResource{"/loadbalancers"}}
}

func (lbs *LoadbalancersDef) Reconcile() {
	log.Println("Reconciling Loadbalancers...")
}

func (lbs *LoadbalancersDef) Write() {
	log.Println("Writing Loadbalancers...")
}

func (lbs *LoadbalancersDef) Sync(f func()) {
	log.Println("Syncing Loadbalancers...")
}
