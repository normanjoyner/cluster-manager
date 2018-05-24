package request

// CloudService indicates the cloud service a request is associated with
type CloudService int

const (
	// CloudServiceAPI is the Cloud API service
	CloudServiceAPI CloudService = iota
	// CloudServiceProvision is the Cloud Provision service
	CloudServiceProvision
)

// String converts this CloudService to its string representation
func (s CloudService) String() string {
	switch s {
	case CloudServiceAPI:
		return "Cloud API Service"
	case CloudServiceProvision:
		return "Cloud Provision Service"
	default:
		return "Unknown"
	}
}
