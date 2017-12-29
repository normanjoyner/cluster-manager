package v3

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	Key         string `json:"key"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// UserList is a list of Users.
type UserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []User `json:"items"`
}
