package resources

import (
	"log"
)

// SSHKeysDef defines the Containership Cloud SSH Keys resource
type SSHKeysDef struct {
	csResource
}

// SSHKeys resource
var SSHKeys *SSHKeysDef

func init() {
	SSHKeys = &SSHKeysDef{csResource{"/ssh-keys"}}
}

// Reconcile compares created SSH Keys against cached SSH Keys
func (sshks *SSHKeysDef) Reconcile() {
	log.Println("Reconciling SSH Keys...")
}

// Write creates SSH Keys on the host
func (sshks *SSHKeysDef) Write() {
	log.Println("Writing SSH Keys...")
}

// Sync fetches SSH Keys from Containership Cloud and executes a callback if the fetched data does not match the internal cache
func (sshks *SSHKeysDef) Sync(f func()) {
	log.Println("Syncing SSH Keys...")
	f()
}
