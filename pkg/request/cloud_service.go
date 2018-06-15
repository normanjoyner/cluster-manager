package request

import (
	"github.com/containership/cloud-agent/pkg/env"
)

// CloudService indicates the cloud service a request is associated with
type CloudService int

const (
	// CloudServiceAPI is the Cloud API service
	CloudServiceAPI CloudService = iota
	// CloudServiceProvision is the Cloud Provision service
	CloudServiceProvision
	// CloudServiceAuth is the Cloud Auth service
	CloudServiceAuth
)

// String converts this CloudService to its string representation
func (s CloudService) String() string {
	switch s {
	case CloudServiceAPI:
		return "Cloud API Service"
	case CloudServiceAuth:
		return "Cloud Auth Service"
	case CloudServiceProvision:
		return "Cloud Provision Service"
	default:
		return "Unknown"
	}
}

// BaseURL returns the url associated with the cloud service
func (s CloudService) BaseURL() string {
	switch s {
	case CloudServiceAPI:
		return env.APIBaseURL()
	case CloudServiceAuth:
		return env.AuthBaseURL()
	case CloudServiceProvision:
		return env.ProvisionBaseURL()
	default:
		return ""
	}
}
