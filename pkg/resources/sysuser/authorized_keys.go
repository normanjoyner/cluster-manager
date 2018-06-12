package sysuser

import (
	"fmt"
	"os"
	"path"
	"sync"

	"github.com/spf13/afero"

	v3 "github.com/containership/cloud-agent/pkg/apis/containership.io/v3"
	"github.com/containership/cloud-agent/pkg/constants"
	"github.com/containership/cloud-agent/pkg/log"
	"github.com/containership/cloud-agent/pkg/tools/fsutil"
)

const (
	// Container
	loginScriptContainerPath = "/scripts/containership_login.sh"

	// Host
	loginScriptFilename       = "containership_login.sh"
	authorizedKeysFilename    = "authorized_keys"
	authorizedKeysPermissions = os.FileMode(0600)
	sshDirPermissions         = os.ModeDir | os.FileMode(0755)
	scriptsDirPermissions     = os.ModeDir | os.FileMode(0755)
	scriptPermissions         = os.FileMode(0755)
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

// GetAuthorizedKeysFullPath returns the full path to the authorized_keys file.
func GetAuthorizedKeysFullPath() string {
	return path.Join(getSSHDir(), authorizedKeysFilename)
}

// InitializeAuthorizedKeysFileStructure does everything required to make SSH
// work on host: creates the SSH directory, initializes a blank authorized_keys
// file, (to simplify e.g. file watching), and puts the login script in place.
// Assumes that ContainershipMount is mounted as a hostPath.
// This is not thread-safe and is expected to only be called on initialization.
func InitializeAuthorizedKeysFileStructure() error {
	return initializeAuthorizedKeysFileStructure(osFs)
}

func initializeAuthorizedKeysFileStructure(fs afero.Fs) error {
	// Create directories / fix permissions if needed
	err := fsutil.EnsureDirExistsWithCorrectPermissions(fs, getSSHDir(), sshDirPermissions)
	if err != nil {
		return err
	}

	err = fsutil.EnsureDirExistsWithCorrectPermissions(fs, getScriptsDir(), scriptsDirPermissions)
	if err != nil {
		return err
	}

	// Create empty authorized_keys if needed (this is just to simplify other
	// logic down the line)
	akFile := GetAuthorizedKeysFullPath()
	if !fsutil.FileExists(fs, akFile) {
		log.Info("authorized_keys file didn't exist so we're creating it")
		f, err := fs.OpenFile(akFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
			authorizedKeysPermissions)
		if err != nil {
			return err
		}
		defer f.Close()
	} else {
		// Ensure permissions of existing file are correct
		if err := fs.Chmod(akFile, sshDirPermissions); err != nil {
			return err
		}
	}

	// Copy login script into place
	err = fsutil.CopyFileForcefully(fs, getLoginScriptFullPath(), loginScriptContainerPath, scriptPermissions)
	if err != nil {
		return err
	}

	return nil
}

// getSSHDir returns the SSH directory built from the environment
func getSSHDir() string {
	return path.Join(constants.ContainershipMount, "home", ".ssh")
}

// getScriptsDir returns the full path to the scripts dir
func getScriptsDir() string {
	return path.Join(constants.ContainershipMount, "scripts")
}

// getLoginScriptFullPath returns the full path to the login script
func getLoginScriptFullPath() string {
	return path.Join(getScriptsDir(), loginScriptFilename)
}

// writeAuthorizedKeys is the same as WriteAuthorizedKeys but takes a
// filesystem argument for testing purposes.
func writeAuthorizedKeys(fs afero.Fs, users []v3.UserSpec) error {
	filename := GetAuthorizedKeysFullPath()

	// Prevent against ssh dir deletion or permissions changes
	err := fsutil.EnsureDirExistsWithCorrectPermissions(fs, getSSHDir(), sshDirPermissions)
	if err != nil {
		return err
	}

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
	if err != nil {
		return err
	}

	// If OpenFile() above didn't create the file, there's a chance the file
	// existed already but with incorrect permissions - fix them.
	return fs.Chmod(filename, authorizedKeysPermissions)
}

// buildKeysStringForUser builds a string containing all authorized_keys lines
// for a single user
func buildKeysStringForUser(user v3.UserSpec) string {
	username := UsernameFromContainershipUID(user.ID)

	// TODO concatenation using + is terribly inefficient
	s := ""
	for _, k := range user.SSHKeys {
		s += fmt.Sprintf("command=\"%s %s\" %s\n",
			getLoginScriptFullPath(), username, k.Key)
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
