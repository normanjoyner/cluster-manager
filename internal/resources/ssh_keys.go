package resources

import (
	"log"
)

type SSHKeysDef struct {
	csResource
}

var SSHKeys *SSHKeysDef

func init() {
	SSHKeys = &SSHKeysDef{csResource{"/ssh-keys"}}
}

func (sshks *SSHKeysDef) Reconcile() {
	log.Println("Reconciling SSH Keys...")
}

func (sshks *SSHKeysDef) Write() {
	log.Println("Writing SSH Keys...")
}

func (sshks *SSHKeysDef) Sync(f func()) {
	log.Println("Syncing SSH Keys...")
	f()
}
