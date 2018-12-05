package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/containership/cluster-manager/pkg/request"
)

func TestGetService(t *testing.T) {
	service := request.CloudServiceAuth

	cr := cloudResource{
		endpoint: "",
		service:  service,
	}

	assert.Equal(t, service, cr.Service())
}

func TestGetEndpoint(t *testing.T) {
	endpoint := "/testing/endpoint"

	cr := cloudResource{
		endpoint: endpoint,
		service:  request.CloudServiceProvision,
	}

	assert.Equal(t, endpoint, cr.Endpoint())
}
