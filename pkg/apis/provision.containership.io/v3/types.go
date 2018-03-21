package v3

import (
	"time"

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
	ID                      string              `json:"id"`
	AddedAt                 string              `json:"addedAt"`
	Description             string              `json:"description"`
	TargetKubernetesVersion string              `json:"targetKubernetesVersion"`
	TargetEtcdVersion       string              `json:"targetEtcdVersion"`
	LabelSelector           []LabelSelectorSpec `json:"labelSelector"`
	Timeout                 string              `json:"timeout"`
	Status                  UpgradeStatus       `json:"status"`
	CurrentNode             string              `json:"currentNode"`
}

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

// NodeUpgradeAnnotation is the upgrade metadata attached to a node
type NodeUpgradeAnnotation struct {
	ClusterVersion string        `json:"clusterVersion"`
	Status         UpgradeStatus `json:"status"`
	StartTime      time.Time     `json:"startTime"`
}

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
