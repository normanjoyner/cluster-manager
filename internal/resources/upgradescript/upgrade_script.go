package upgradescript

import (
	"fmt"
	"os"
	"path"

	"github.com/spf13/afero"

	"github.com/containership/cloud-agent/internal/constants"
	"github.com/containership/cloud-agent/internal/log"
	"github.com/containership/cloud-agent/internal/tools/fsutil"
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
func Write(script []byte, targetVersion, upgradeID string) error {
	filename := GetUpgradeScriptFullPath(getUpgradeScriptFilename(targetVersion, upgradeID))
	return writeUpgradeScript(osFs, script, filename)
}

// RemoveCurrent removes the /current file after an upgrade has completed
func RemoveCurrent() error {
	return removeCurrentUpgradeScript(osFs)
}

// writeUpgradeScript takes the given script and writes it to the path
// that systemd is watching for it to be ran on the host
func writeUpgradeScript(fs afero.Fs, script []byte, upgradeScriptFilename string) error {
	currentScriptFilename := GetUpgradeScriptFullPath(currentScriptFile)

	err := fsutil.EnsureDirExistsWithCorrectPermissions(fs, getScriptDir(), upgradeScriptDirPermissions)
	if err != nil {
		return err
	}

	err = writeScript(fs, upgradeScriptFilename, script)
	if err != nil {
		return err
	}

	err = writePathToCurrent(fs, currentScriptFilename, upgradeScriptFilename)
	if err != nil {
		return err
	}

	return nil
}

// writeScript takes and writes the upgrade script to a named file with
// the target version, and id of the upgrade
func writeScript(fs afero.Fs, filename string, script []byte) error {
	return write(fs, filename, upgradeScriptPermissions, script)
}

// writePathToCurrent takes the filename to /current, and the filename to the
// upgrade script that needs to be ran
func writePathToCurrent(fs afero.Fs, filename, upgradeScriptPath string) error {
	return write(fs, filename, currentUpgradeScriptPermissions, []byte(upgradeScriptPath))
}

// write takes data and writes it to the specified file with the correct permissions
func write(fs afero.Fs, filename string, permissions os.FileMode, script []byte) error {
	// Checks to see if the upgrade script already exists so it doesn't get
	// overwritten, if its in the process of running
	if fsutil.FileExists(fs, filename) {
		return fmt.Errorf("upgrade script %s already exists", filename)
	}

	// Create the upgrade script to write to
	f, err := fs.OpenFile(filename, os.O_CREATE|os.O_WRONLY, permissions)

	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(script)
	if err != nil {
		return err
	}

	return nil
}

// removeCurrentUpgradeScript removes the /current file after an upgrade
// has completed
func removeCurrentUpgradeScript(fs afero.Fs) error {
	currentScriptFilename := GetUpgradeScriptFullPath(currentScriptFile)

	log.Debug("Upgrade file \"current\" is being removed.")
	return fs.Remove(currentScriptFilename)
}

// GetUpgradeScriptFullPath returns path to the upgrade script
func GetUpgradeScriptFullPath(filename string) string {
	return path.Join(getScriptDir(), filename)
}

// getScriptDir returns the directory to top level directory for upgrades
func getScriptDir() string {
	return path.Join(constants.ContainershipMount, upgradeScriptDir)
}

// getUpgradeScriptFilename returns the file name that will be used for upgrade
func getUpgradeScriptFilename(targetVersion, upgradeID string) string {
	return fmt.Sprintf("upgrade-%s-%s.sh", targetVersion, upgradeID)
}
