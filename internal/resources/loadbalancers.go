package resources

import (
	"github.com/containership/cloud-agent/internal/log"
)

// Loadbalancers defines the Containership Cloud Loadbalancers resource
type Loadbalancers struct {
	csResource
}

// NewLoadbalancers constructs a new Loadbalancers
func NewLoadbalancers() *Loadbalancers {
	return &Loadbalancers{csResource{
		Endpoint: "/organizations/{{.OrganizationID}}/clusters/{{.ClusterID}}/loadbalancers",
		Type:     ResourceTypeCluster,
	}}
}

// GetEndpoint returns the Endpoint
func (lbs *Loadbalancers) GetEndpoint() string {
	return lbs.Endpoint
}

// GetType returns the ResourceType
func (lbs *Loadbalancers) GetType() ResourceType {
	return lbs.Type
}

// Reconcile compares created loadbalancers against cached loadbalancers
func (lbs *Loadbalancers) Reconcile() {
	log.Info("Reconciling Loadbalancers...")
}

// Sync fetches loadbalancers from Containership Cloud and executes a callback
// if the fetched data does not match the internal cache
func (lbs *Loadbalancers) Sync(onCacheMismatch func()) error {
	log.Info("Syncing Loadbalancers...")
	onCacheMismatch()
	return nil
}

// Write creates loadbalancers on the cluster
func (lbs *Loadbalancers) Write() {
	log.Info("Writing Loadbalancers...")
}
