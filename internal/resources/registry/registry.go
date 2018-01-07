package registry

import (
	"encoding/base64"
	"strings"

	containershipv3 "github.com/containership/cloud-agent/pkg/apis/containership.io/v3"
)

const (
	// EC2Registry is the name of an amazon registry in containership cloud
	EC2Registry = "amazon_ec2_registry"
	// Azure is the name of an azure registry in containership cloud
	Azure = "azure"
	// Docker is the name of a docker registry in containership cloud
	Docker = "dockerhub"
	// GCR is the name of a google cloud registry in containership cloud
	GCR = "google_registry"
	// Private is the name of a private registry in containership cloud
	Private = "private"
	// Quay is the name of a quay registry in containership cloud
	Quay = "quay"
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

	if provider == EC2Registry {
		g = ECR{credentials}
	} else {
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
		Type:     DockerCFG,
	}, nil
}
