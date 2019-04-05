package resources

import (
	"encoding/json"

	"github.com/pkg/errors"

	cscloud "github.com/containership/csctl/cloud"
	"github.com/containership/csctl/cloud/provision/types"

	csv3 "github.com/containership/cluster-manager/pkg/apis/provision.containership.io/v3"
)

// CsNodePoolLabels defines the Containership Cloud NodePoolLabels resource
type CsNodePoolLabels struct {
	cloudResource
	cache []csv3.NodePoolLabelSpec
}

// NewCsNodePoolLabels constructs a new CsNodePoolLabels
func NewCsNodePoolLabels(cloud cscloud.Interface) *CsNodePoolLabels {
	cache := make([]csv3.NodePoolLabelSpec, 0)
	return &CsNodePoolLabels{
		newCloudResource(cloud),
		cache,
	}
}

// Sync implements the CloudResource interface
func (l *CsNodePoolLabels) Sync() error {
	pools, err := l.cloud.Provision().NodePools(l.organizationID, l.clusterID).List()
	if err != nil {
		return errors.Wrap(err, "listing node pools")
	}

	nodePoolLabels := make([]types.NodePoolLabel, 0)
	for _, pool := range pools {
		labels, err := l.cloud.Provision().NodePoolLabels(l.organizationID, l.clusterID, string(pool.ID)).List()
		if err != nil {
			return errors.Wrapf(err, "listing node pool labels for node pool %q", pool.ID)
		}

		nodePoolLabels = append(nodePoolLabels, labels...)
	}

	data, err := json.Marshal(nodePoolLabels)
	if err != nil {
		return errors.Wrap(err, "marshaling node pool labels")
	}

	json.Unmarshal(data, &l.cache)

	return nil
}

// Cache return the containership nodePoolLabels cache
func (l *CsNodePoolLabels) Cache() []csv3.NodePoolLabelSpec {
	return l.cache
}

// IsEqual compares a NodePoolLabelSpec to another NodePoolLabel
func (l *CsNodePoolLabels) IsEqual(specObj interface{}, parentSpecObj interface{}) (bool, error) {
	spec, ok := specObj.(csv3.NodePoolLabelSpec)
	if !ok {
		return false, errors.New("object is not of type NodePoolLabelSpec")
	}

	nodePoolLabel, ok := parentSpecObj.(*csv3.NodePoolLabel)
	if !ok {
		return false, errors.New("object is not of type NodePoolLabel")
	}

	equal := spec.ID == nodePoolLabel.Spec.ID &&
		spec.CreatedAt == nodePoolLabel.Spec.CreatedAt &&
		spec.UpdatedAt == nodePoolLabel.Spec.UpdatedAt &&
		spec.Key == nodePoolLabel.Spec.Key &&
		spec.Value == nodePoolLabel.Spec.Value &&
		spec.NodePoolID == nodePoolLabel.Spec.NodePoolID

	return equal, nil
}
