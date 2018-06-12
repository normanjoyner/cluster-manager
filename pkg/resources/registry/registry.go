package registry

import (
	"encoding/base64"
	"strings"

	containershipv3 "github.com/containership/cloud-agent/pkg/apis/containership.io/v3"

	"github.com/containership/cloud-agent/pkg/constants"
)

const (
	// DockerCFG lets registries know its token, and endpoint should be used in
	// the format of a ~/.dockercfg file
	DockerCFG string = "dockercfg"
	// DockerJSON lets registries know its token, and endpoint should be used
	// in the format of a ~/.docker/config.json file
	DockerJSON string = "dockerconfigjson"
)

// Generator allows us to access and return registry logic
// in the same way for all registries
type Generator interface {
	CreateAuthToken() (containershipv3.AuthTokenDef, error)
}

// Default is the type of registry if they don't need special logic
type Default struct {
	Credentials map[string]string
	endpoint    string
}

// New returns a Generator of the correct type to be able to get the
// AuthToken for the registry
func New(provider, endpoint string, credentials map[string]string) Generator {
	var g Generator

	switch provider {
	case constants.EC2Registry:
		g = ECR{credentials}
	case constants.GCR:
		g = GCR{Default{credentials, endpoint}}
	default:
		g = Default{credentials, endpoint}
	}

	return g
}

// Username returns the registry username that should be used for auth
func (d Default) Username() string {
	return d.Credentials["username"]
}

// Password returns the registry password that should be used for auth
func (d Default) Password() string {
	return d.Credentials["password"]
}

// Endpoint returns the endpoint that should be used for the container registry
func (d Default) Endpoint() string {
	return d.endpoint
}

// CreateAuthToken returns a base64 encrypted token to use as an Auth token
func (d Default) CreateAuthToken() (containershipv3.AuthTokenDef, error) {
	return containershipv3.AuthTokenDef{
		Token:    base64.StdEncoding.EncodeToString([]byte(strings.Join([]string{d.Username(), d.Password()}, ":"))),
		Endpoint: d.Endpoint(),
		Type:     DockerJSON,
	}, nil
}
