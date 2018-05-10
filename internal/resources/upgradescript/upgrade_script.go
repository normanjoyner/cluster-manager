package upgradescript

import (
	"fmt"
	"os"
	"path"

	"github.com/spf13/afero"

	"github.com/containership/cloud-agent/internal/constants"
	"github.com/containership/cloud-agent/internal/log"
	"github.com/containership/cloud-agent/internal/tools/fsutil"

	provisioncsv3 "github.com/containership/cloud-agent/pkg/apis/provision.containership.io/v3"
)

const (
	// Host filesystem paths (relative to containership mount)
	upgradeScriptDir  = "/scripts/upgrade"
	currentScriptFile = "current"

	// TODO discuss how we want to restructure the containership area more
	// and what the correct permissions for this file should be
	upgradeScriptPermissions        = os.FileMode(0700)
	currentUpgradeScriptPermissions = os.FileMode(0600)
	upgradeScriptDirPermissions     = os.ModeDir | os.FileMode(0700)
)

// This is a no-op and always references the same underlying OS filesystem, so
// it's fine to do it any file that has file operations that we'd like to make
// testable.
var osFs = afero.NewOsFs()

// Write takes the script data to write to /upgrade/upgradeScriptFilename
func Write(script []byte, upgradeType provisioncsv3.UpgradeType, targetVersion, upgradeID string) error {
	filename := GetUpgradeScriptFullPath(getUpgradeScriptFilename(upgradeType, targetVersion, upgradeID))
	return writeUpgradeScript(osFs, script, filename)
}

// Exists returns true if the upgrade script exists on disk (i.e.
// Write() has been called for this upgrade), else false
func Exists(upgradeType provisioncsv3.UpgradeType, targetVersion, upgradeID string) bool {
	return exists(osFs, upgradeType, targetVersion, upgradeID)
}

// GetUpgradeScriptFullPath returns path to the upgrade script
func GetUpgradeScriptFullPath(filename string) string {
	return path.Join(getScriptDir(), filename)
}

// RemoveCurrent removes the /current file after an upgrade has completed
func RemoveCurrent() error {
	return removeCurrent(osFs)
}

func exists(fs afero.Fs, upgradeType provisioncsv3.UpgradeType, targetVersion, upgradeID string) bool {
	filename := GetUpgradeScriptFullPath(getUpgradeScriptFilename(upgradeType, targetVersion, upgradeID))

	return fsutil.FileExists(fs, filename)
}

// removeCurrent removes the /current file after an upgrade
// has completed. nil is returned if the file does not exist.
func removeCurrent(fs afero.Fs) error {
	currentScriptFilename := GetUpgradeScriptFullPath(currentScriptFile)

	log.Debug("Upgrade file \"current\" is being removed.")
	// RemoveAll instead of Remove so no error is returned if `current` DNE.
	return fs.RemoveAll(currentScriptFilename)
}

// writeUpgradeScript takes the given script and writes it to the path
// that systemd is watching for it to be ran on the host
func writeUpgradeScript(fs afero.Fs, script []byte, upgradeScriptFilename string) error {
	err := fsutil.EnsureDirExistsWithCorrectPermissions(fs, getScriptDir(), upgradeScriptDirPermissions)
	if err != nil {
		return err
	}

	err = writeScript(fs, upgradeScriptFilename, script)
	if err != nil {
		return err
	}

	err = writePathToCurrent(fs, upgradeScriptFilename)
	if err != nil {
		return err
	}

	return nil
}

// writeScript takes and writes the upgrade script to a named file with
// the target version, and id of the upgrade
func writeScript(fs afero.Fs, filename string, script []byte) error {
	return fsutil.WriteNewFile(fs, filename, script, upgradeScriptPermissions)
}

// writePathToCurrent writes the current upgrade script path to the `current` file
// at its standard location
func writePathToCurrent(fs afero.Fs, upgradeScriptPath string) error {
	currentPath := GetUpgradeScriptFullPath(currentScriptFile)
	return fsutil.WriteNewFileAtomic(fs, currentPath, []byte(upgradeScriptPath), currentUpgradeScriptPermissions)
}

// getScriptDir returns the directory to top level directory for upgrades
func getScriptDir() string {
	return path.Join(constants.ContainershipMount, upgradeScriptDir)
}

// getUpgradeScriptFilename returns the file name that will be used for upgrade
func getUpgradeScriptFilename(upgradeType provisioncsv3.UpgradeType, targetVersion, upgradeID string) string {
	return fmt.Sprintf("upgrade-%s-%s-%s.sh", upgradeType, targetVersion, upgradeID)
}
