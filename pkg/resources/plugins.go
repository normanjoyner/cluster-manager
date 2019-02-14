package resources

import (
	"encoding/json"
	"fmt"

	cscloud "github.com/containership/csctl/cloud"

	csv3 "github.com/containership/cluster-manager/pkg/apis/containership.io/v3"
)

// CsPlugins defines the Containership Cloud Plugins resource
type CsPlugins struct {
	cloudResource
	cache []csv3.PluginSpec
}

// NewCsPlugins constructs a new CsPlugins
func NewCsPlugins(cloud cscloud.Interface) *CsPlugins {
	cache := make([]csv3.PluginSpec, 0)
	return &CsPlugins{
		newCloudResource(cloud),
		cache,
	}
}

// Sync implements the CloudResource interface
func (cp *CsPlugins) Sync() error {
	plugins, err := cp.cloud.API().Plugins(cp.organizationID, cp.clusterID).List()
	if err != nil {
		return err
	}

	data, err := json.Marshal(plugins)
	if err != nil {
		return err
	}

	json.Unmarshal(data, &cp.cache)

	return nil
}

// Cache return the containership plugins cache
func (cp *CsPlugins) Cache() []csv3.PluginSpec {
	return cp.cache
}

// IsEqual compares a PluginSpec to another Plugin
func (cp *CsPlugins) IsEqual(specObj interface{}, parentSpecObj interface{}) (bool, error) {
	spec, ok := specObj.(csv3.PluginSpec)
	if !ok {
		return false, fmt.Errorf("The object is not of type PluginSpec")
	}

	plugin, ok := parentSpecObj.(*csv3.Plugin)
	if !ok {
		return false, fmt.Errorf("The object is not of type Plugin")
	}

	if plugin.Spec.Version != spec.Version {
		return false, nil
	}

	return true, nil
}
