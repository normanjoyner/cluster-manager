package resources

import (
	cscloud "github.com/containership/csctl/cloud"

	"github.com/containership/cluster-manager/pkg/env"
	"github.com/containership/cluster-manager/pkg/request"
)

// CloudResource defines an interface for resources to adhere to in order to be kept
// in sync with Containership Cloud
type CloudResource interface {
	// IsEqual compares a spec to it's parent object spec
	IsEqual(spec interface{}, parentSpecObj interface{}) (bool, error)
	// Service returns the request.CloudService type of the API to make a request to
	Service() request.CloudService
	// Sync grabs its resources from Containership Cloud and writes them to cache
	Sync() error
}

// cloudResource defines what each resource needs to contain
type cloudResource struct {
	cloud          cscloud.Interface
	organizationID string
	clusterID      string
}

func newCloudResource(cloud cscloud.Interface) cloudResource {
	return cloudResource{
		cloud:          cloud,
		organizationID: env.OrganizationID(),
		clusterID:      env.ClusterID(),
	}
}
