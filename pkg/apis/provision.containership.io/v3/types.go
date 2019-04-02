package v3

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/selection"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterUpgrade describes the cluster upgrade that has been requested.
// This is not synced from cloud
type ClusterUpgrade struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ClusterUpgradeSpec `json:"spec"`
}

// ClusterUpgradeSpec is the spec for a Containership Cloud Cluster Upgrade.
type ClusterUpgradeSpec struct {
	ID                 string                   `json:"id"`
	Type               UpgradeType              `json:"type"`
	AddedAt            string                   `json:"addedAt"`
	Description        string                   `json:"description"`
	TargetVersion      string                   `json:"targetVersion"`
	LabelSelector      []LabelSelectorSpec      `json:"labelSelector"`
	NodeTimeoutSeconds int                      `json:"nodeTimeoutSeconds"`
	Status             ClusterUpgradeStatusSpec `json:"status"`
}

// ClusterUpgradeStatusSpec is the spec for the current status / state of a
// Cluster Upgrade. It is never modified by Cloud.
type ClusterUpgradeStatusSpec struct {
	ClusterStatus    UpgradeStatus            `json:"clusterStatus"`
	NodeStatuses     map[string]UpgradeStatus `json:"nodeStatuses"`
	CurrentNode      string                   `json:"currentNode"`
	CurrentStartTime string                   `json:"currentStartTime"`
}

// UpgradeType specifies the type of upgrade this CRD corresponds to
type UpgradeType string

const (
	// UpgradeTypeKubernetes is for upgrading Kubernetes
	UpgradeTypeKubernetes UpgradeType = "kubernetes"
	// UpgradeTypeEtcd is for upgrading etcd (not yet supported)
	UpgradeTypeEtcd UpgradeType = "etcd"
)

// UpgradeStatus keeps track of where in the upgrade process the cluster is
type UpgradeStatus string

const (
	// UpgradeInProgress means the update process has started
	UpgradeInProgress UpgradeStatus = "InProgress"
	// UpgradeSuccess status gets set when all nodes have been updated to Target Version
	UpgradeSuccess UpgradeStatus = "Success"
	// UpgradeFailed status gets set when 1 or more nodes in upgrade if unsuccessful
	UpgradeFailed UpgradeStatus = "Failed"
)

// LabelSelectorSpec lets a user add more filters to the nodes they want to update
type LabelSelectorSpec struct {
	Label    string             `json:"label"`
	Operator selection.Operator `json:"operator"`
	Value    []string           `json:"value"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterUpgradeList is a list of ClusterUpgrades.
type ClusterUpgradeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []ClusterUpgrade `json:"items"`
}

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodePoolLabel describes a node pool label in Containership Cloud.
type NodePoolLabel struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec NodePoolLabelSpec `json:"spec"`
}

// NodePoolLabelSpec is the spec for a Containership Cloud NodePoolLabel.
type NodePoolLabelSpec struct {
	ID         string `json:"id"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
	Key        string `json:"key"`
	Value      string `json:"value"`
	NodePoolID string `json:"node_pool_id"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodePoolLabelList is a list of NodePoolLabels.
type NodePoolLabelList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []NodePoolLabel `json:"items"`
}
