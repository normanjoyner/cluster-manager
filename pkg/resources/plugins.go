package resources

import (
	"encoding/json"
	"fmt"

	containershipv3 "github.com/containership/cloud-agent/pkg/apis/containership.io/v3"
)

// CsPlugins defines the Containership Cloud Plugins resource
type CsPlugins struct {
	cloudResource
	cache []containershipv3.PluginSpec
}

// NewCsPlugins constructs a new CsPlugins
func NewCsPlugins() *CsPlugins {
	return &CsPlugins{
		cloudResource: cloudResource{
			endpoint: "/organizations/{{.OrganizationID}}/clusters/{{.ClusterID}}/plugins",
		},
		cache: make([]containershipv3.PluginSpec, 0),
	}
}

// UnmarshalToCache take the json returned from containership api
// and writes it to CsPlugins cache
func (us *CsPlugins) UnmarshalToCache(bytes []byte) error {
	return json.Unmarshal(bytes, &us.cache)
}

// Cache return the containership plugins cache
func (us *CsPlugins) Cache() []containershipv3.PluginSpec {
	return us.cache
}

// IsEqual compares a PluginSpec to another Plugin
func (us *CsPlugins) IsEqual(specObj interface{}, parentSpecObj interface{}) (bool, error) {
	spec, ok := specObj.(containershipv3.PluginSpec)
	if !ok {
		return false, fmt.Errorf("The object is not of type PluginSpec")
	}

	plugin, ok := parentSpecObj.(*containershipv3.Plugin)
	if !ok {
		return false, fmt.Errorf("The object is not of type Plugin")
	}

	if plugin.Spec.Version != spec.Version {
		return false, nil
	}

	return true, nil
}
