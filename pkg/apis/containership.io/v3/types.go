package v3

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/selection"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// User describes a Containership Cloud user.
type User struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec UserSpec `json:"spec"`
}

// UserSpec is the spec for a Containership Cloud user.
type UserSpec struct {
	ID        string       `json:"id"`
	Name      string       `json:"name"`
	AvatarURL string       `json:"avatar_url"`
	AddedAt   string       `json:"added_at"`
	SSHKeys   []SSHKeySpec `json:"ssh_keys"`
}

// SSHKeySpec is the spec for an SSH Key.
type SSHKeySpec struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Fingerprint string `json:"fingerprint"`
	Key         string `json:"key"` // format: "<key_type> <key>"
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// UserList is a list of Users.
type UserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []User `json:"items"`
}

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Registry describes a registry attached to Containership Cloud.
type Registry struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec RegistrySpec `json:"spec"`
}

// RegistrySpec is the spec for a Containership Cloud Registry.
type RegistrySpec struct {
	ID            string            `json:"id"`
	AddedAt       string            `json:"added_at"`
	Description   string            `json:"description"`
	Organization  string            `json:"organization_id"`
	Email         string            `json:"email"`
	Serveraddress string            `json:"serveraddress"`
	Provider      string            `json:"provider"`
	Credentials   map[string]string `json:"credentials"`
	Owner         string            `json:"owner"`
	AuthToken     AuthTokenDef      `json:"authToken,omitempty"`
}

// AuthTokenDef is the def for an auth token
type AuthTokenDef struct {
	Token    string `json:"token"`
	Endpoint string `json:"endpoint"`
	Type     string `json:"type"`
	Expires  string `json:"expires"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RegistryList is a list of Registries.
type RegistryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Registry `json:"items"`
}

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Plugin describes a plugin added by Containership Cloud.
type Plugin struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec PluginSpec `json:"spec"`
}

// PluginSpec is the spec for a Containership Cloud Plugin.
type PluginSpec struct {
	ID             string     `json:"id"`
	AddedAt        string     `json:"added_at"`
	Description    string     `json:"description"`
	Type           PluginType `json:"type"`
	Version        string     `json:"version"`
	Implementation string     `json:"implementation"`
}

// PluginType lets us group together plugins of different implentations
type PluginType string

const (
	// Logs is a generic type of supported plugin
	Logs PluginType = "logs"
	// Metrics is a generic type of supported plugin
	Metrics PluginType = "metrics"
	// Events is a generic type of event supported plugin
	Events PluginType = "events"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PluginList is a list of Plugins.
type PluginList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Plugin `json:"items"`
}

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
	AddedAt                 string              `json:"added_at"`
	Description             string              `json:"description"`
	TargetKubernetesVersion string              `json:"target_kubernetes_version"`
	TargetEtcdVersion       string              `json:"target_etcd_version"`
	LabelSelector           []LabelSelectorSpec `json:"label_selector"`
	Timeout                 string              `json:"timeout"`
	Status                  UpgradeStatus       `json:"status"`
	CurrentNode             string              `json:"current_node"`
}

// UpgradeStatus keeps track of where in the upgrade process the cluster is
type UpgradeStatus int

const (
	// UpgradeInProgress means the update process has started
	UpgradeInProgress UpgradeStatus = 1
	// UpgradeSuccess status gets set when all nodes have been updated to Target Version
	UpgradeSuccess UpgradeStatus = 2
	// UpgradeFailed status gets set when 1 or more nodes in upgrade if unsucessful
	UpgradeFailed UpgradeStatus = 3
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
