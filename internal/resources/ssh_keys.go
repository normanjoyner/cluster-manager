package resources

import (
	"log"
)

// SSHKeys defines the Containership Cloud SSH Keys resource
type SSHKeys struct {
	csResource
}

// NewSSHKeys constructs a new SSHKeys
func NewSSHKeys() *SSHKeys {
	return &SSHKeys{csResource{
		Endpoint: "/organizations/{{.OrganizationID}}/ssh_keys",
		Type:     ResourceTypeHost,
	}}
}

// GetEndpoint returns the Endpoint
func (sshks *SSHKeys) GetEndpoint() string {
	return sshks.Endpoint
}

// GetType returns the ResourceType
func (sshks *SSHKeys) GetType() ResourceType {
	return ResourceTypeHost
}

// Reconcile compares created SSH Keys against cached SSH Keys
func (sshks *SSHKeys) Reconcile() {
	log.Println("Reconciling SSH Keys...")
}

// Sync fetches SSH Keys from Containership Cloud and executes a callback if the fetched data does not match the internal cache
func (sshks *SSHKeys) Sync(onCacheMismatch func()) error {
	log.Println("Syncing SSH Keys...")
	onCacheMismatch()
	return nil
}

// Write creates SSH Keys on the host
func (sshks *SSHKeys) Write() {
	log.Println("Writing SSH Keys...")
}
