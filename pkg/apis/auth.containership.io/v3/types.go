package v3

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AuthorizationRole describes an authorization role in Containership Cloud.
type AuthorizationRole struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec AuthorizationRoleSpec `json:"spec"`
}

// AuthorizationRoleSpec is the spec for a Containership Cloud auth role.
type AuthorizationRoleSpec struct {
	ID             string                  `json:"id"`
	CreatedAt      string                  `json:"created_at"`
	UpdatedAt      string                  `json:"updated_at"`
	OrganizationID string                  `json:"organization_id"`
	Name           string                  `json:"name"`
	Description    string                  `json:"description,omitempty"`
	OwnerID        string                  `json:"owner_id"`
	Rules          []AuthorizationRuleSpec `json:"rules"`
}

// AuthorizationRuleSpec is the spec for a Containership Cloud auth rule.
type AuthorizationRuleSpec struct {
	ID              string                `json:"id"`
	CreatedAt       string                `json:"created_at"`
	UpdatedAt       string                `json:"updated_at"`
	OrganizationID  string                `json:"organization_id"`
	Name            string                `json:"name"`
	Description     string                `json:"description,omitempty"`
	OwnerID         string                `json:"owner_id"`
	Type            AuthorizationRuleType `json:"type"`
	APIGroups       []string              `json:"api_groups,omitempty"`
	Resources       []string              `json:"resources"`
	ResourceNames   []string              `json:"resource_names"`
	Verbs           []string              `json:"verbs"`
	NonResourceURLs []string              `json:"non_resource_urls"`
}

// AuthorizationRuleType specifies the type of rule
type AuthorizationRuleType string

const (
	// AuthorizationRuleTypeKubernetes applies to Kubernetes
	AuthorizationRuleTypeKubernetes AuthorizationRuleType = "kubernetes"
	// AuthorizationRuleTypeContainership applies to Containership Cloud
	AuthorizationRuleTypeContainership AuthorizationRuleType = "containership"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AuthorizationRoleList is a list of AuthorizationRoles.
type AuthorizationRoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []AuthorizationRole `json:"items"`
}

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AuthorizationRoleBinding describes an authorization role binding in Containership Cloud.
type AuthorizationRoleBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec AuthorizationRoleBindingSpec `json:"spec"`
}

// AuthorizationRoleBindingSpec is the spec for a Containership Cloud auth role binding.
type AuthorizationRoleBindingSpec struct {
	ID                  string                       `json:"id"`
	CreatedAt           string                       `json:"created_at"`
	UpdatedAt           string                       `json:"updated_at"`
	OrganizationID      string                       `json:"organization_id"`
	OwnerID             string                       `json:"owner_id"`
	Type                AuthorizationRoleBindingType `json:"type"`
	AuthorizationRoleID string                       `json:"authorization_role_id"`

	// The following are conditionally required depending on the Type
	UserID    string `json:"user_id,omitempty"`
	TeamID    string `json:"team_id,omitempty"`
	ClusterID string `json:"cluster_id,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

// AuthorizationRoleBindingType specifies the type of rule
type AuthorizationRoleBindingType string

const (
	// AuthorizationRoleBindingTypeUser binds a role to an individual user
	AuthorizationRoleBindingTypeUser AuthorizationRoleBindingType = "UserBinding"
	// AuthorizationRoleBindingTypeTeam binds a role to an entire team
	AuthorizationRoleBindingTypeTeam AuthorizationRoleBindingType = "TeamBinding"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AuthorizationRoleBindingList is a list of AuthorizationRoleBindings.
type AuthorizationRoleBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []AuthorizationRoleBinding `json:"items"`
}
