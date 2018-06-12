package registry

import (
	"encoding/json"
	"strings"

	containershipv3 "github.com/containership/cloud-agent/pkg/apis/containership.io/v3"
)

// GCR is a google container registry which needs a different kind of auth token
// created to work as an image pull secret
type GCR struct {
	Default
}

// CreateAuthToken returns a base64 encrypted token to use as an Auth token
func (g GCR) CreateAuthToken() (containershipv3.AuthTokenDef, error) {
	token, err := json.Marshal(g.Credentials)
	if err != nil {
		return containershipv3.AuthTokenDef{}, err
	}

	return containershipv3.AuthTokenDef{
		Token:    strings.Replace(string(token), `"`, `\"`, -1),
		Endpoint: "https://" + g.Endpoint(),
		Type:     DockerCFG,
	}, nil
}
