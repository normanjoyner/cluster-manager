package sysuser

import (
	"fmt"
	"os"

	"github.com/spf13/afero"

	v3 "github.com/containership/cloud-agent/pkg/apis/containership.io/v3"
)

const (
	// TODO env vars
	loginScript               = "/opt/containership/home/containership_login.sh"
	authorizedKeysFile        = "/opt/containership/home/.ssh/authorized_keys"
	authorizedKeysPermissions = 0600
)

// This is a no-op and always references the same underlying OS filesystem, so
// it's fine to do it any file that has file operations that we'd like to make
// testable.
var osFs = afero.NewOsFs()

// WriteAuthorizedKeys writes the authorized keys for the given users to the
// authorized_keys file, stomping on the existing file.
func WriteAuthorizedKeys(users []v3.UserSpec) error {
	return writeAuthorizedKeys(osFs, users)
}

// writeAuthorizedKeys is the same as WriteAuthorizedKeys but takes a
// filesystem argument for testing purposes.
func writeAuthorizedKeys(fs afero.Fs, users []v3.UserSpec) error {
	f, err := fs.OpenFile(authorizedKeysFile, os.O_CREATE|os.O_WRONLY,
		authorizedKeysPermissions)
	if err != nil {
		return err
	}
	defer f.Close()

	s := buildAllKeysString(users)

	_, err = f.Write([]byte(s))

	return err
}

// buildKeysStringForUser builds a string containing all authorized_keys lines
// for a single user
func buildKeysStringForUser(user v3.UserSpec) string {
	username := UsernameFromContainershipUID(user.ID)

	// TODO concatenation using + is terribly inefficient
	s := ""
	for _, k := range user.SSHKeys {
		s += fmt.Sprintf("command=\"%s %s\" %s\n",
			loginScript, username, k.Key)
	}

	return s
}

// buildAllKeysString builds the entire authorized_keys contents into a string
func buildAllKeysString(users []v3.UserSpec) string {
	// TODO concatenation using + is terribly inefficient
	s := ""
	for _, u := range users {
		s += buildKeysStringForUser(u)
	}
	return s
}
