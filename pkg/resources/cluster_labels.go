package resources

import (
	"encoding/json"

	"github.com/pkg/errors"

	cscloud "github.com/containership/csctl/cloud"

	csv3 "github.com/containership/cluster-manager/pkg/apis/containership.io/v3"
)

// CsClusterLabels defines the Containership Cloud ClusterLabels resource
type CsClusterLabels struct {
	cloudResource
	cache []csv3.ClusterLabelSpec
}

// NewCsClusterLabels constructs a new CsClusterLabels
func NewCsClusterLabels(cloud cscloud.Interface) *CsClusterLabels {
	cache := make([]csv3.ClusterLabelSpec, 0)
	return &CsClusterLabels{
		newCloudResource(cloud),
		cache,
	}
}

// Sync implements the CloudResource interface
func (l *CsClusterLabels) Sync() error {
	clusterLabels, err := l.cloud.API().ClusterLabels(l.organizationID, l.clusterID).List()
	if err != nil {
		return errors.Wrap(err, "listing cluster labels")
	}

	data, err := json.Marshal(clusterLabels)
	if err != nil {
		return errors.Wrap(err, "marshaling cluster labels")
	}

	json.Unmarshal(data, &l.cache)

	return nil
}

// Cache return the containership clusterLabels cache
func (l *CsClusterLabels) Cache() []csv3.ClusterLabelSpec {
	return l.cache
}

// IsEqual compares a ClusterLabelSpec to another ClusterLabel
func (l *CsClusterLabels) IsEqual(specObj interface{}, parentSpecObj interface{}) (bool, error) {
	spec, ok := specObj.(csv3.ClusterLabelSpec)
	if !ok {
		return false, errors.New("object is not of type ClusterLabelSpec")
	}

	clusterLabel, ok := parentSpecObj.(*csv3.ClusterLabel)
	if !ok {
		return false, errors.New("object is not of type ClusterLabel")
	}

	equal := spec.ID == clusterLabel.Spec.ID &&
		spec.CreatedAt == clusterLabel.Spec.CreatedAt &&
		spec.UpdatedAt == clusterLabel.Spec.UpdatedAt &&
		spec.Key == clusterLabel.Spec.Key &&
		spec.Value == clusterLabel.Spec.Value

	return equal, nil
}
