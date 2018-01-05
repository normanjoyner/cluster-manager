package sysuser

import (
	"fmt"
	"os"
	"path"
	"sync"

	"github.com/spf13/afero"

	"github.com/containership/cloud-agent/internal/envvars"
	v3 "github.com/containership/cloud-agent/pkg/apis/containership.io/v3"
)

const (
	// TODO where does this stuff actually belong?
	loginScriptFilename       = "containership_login.sh"
	authorizedKeysFilename    = "authorized_keys"
	authorizedKeysPermissions = 0600
)

// This is a no-op and always references the same underlying OS filesystem, so
// it's fine to do it any file that has file operations that we'd like to make
// testable.
var osFs = afero.NewOsFs()

// This mutex protects against simultaneous write attempts to authorized_keys.
// Note that the zero-value for a mutex is unlocked.
var writeMutex sync.Mutex

// WriteAuthorizedKeys writes the authorized keys for the given users to the
// authorized_keys file, stomping on the existing file.
func WriteAuthorizedKeys(users []v3.UserSpec) error {
	return writeAuthorizedKeys(osFs, users)
}

// writeAuthorizedKeys is the same as WriteAuthorizedKeys but takes a
// filesystem argument for testing purposes.
func writeAuthorizedKeys(fs afero.Fs, users []v3.UserSpec) error {
	filename := buildAuthorizedKeysFullPath()

	s := buildAllKeysString(users)

	writeMutex.Lock()
	defer writeMutex.Unlock()

	// O_TRUNC so we clear the file contents if there are no keys to write
	f, err := fs.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
		authorizedKeysPermissions)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write([]byte(s))

	return err
}

// buildKeysStringForUser builds a string containing all authorized_keys lines
// for a single user
func buildKeysStringForUser(user v3.UserSpec) string {
	loginScriptFullPath := path.Join(envvars.GetCSHome(), loginScriptFilename)
	username := UsernameFromContainershipUID(user.ID)

	// TODO concatenation using + is terribly inefficient
	s := ""
	for _, k := range user.SSHKeys {
		s += fmt.Sprintf("command=\"%s %s\" %s\n",
			loginScriptFullPath, username, k.Key)
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

func buildAuthorizedKeysFullPath() string {
	return path.Join(envvars.GetCSHome(), ".ssh", authorizedKeysFilename)
}
