package fsutil

import (
	"io"
	"os"

	"github.com/spf13/afero"

	"github.com/containership/cloud-agent/internal/log"
)

// dirExists returns true if dir exists and is a directory, else false
func dirExists(fs afero.Fs, dir string) bool {
	stat, err := fs.Stat(dir)
	if err != nil {
		return false
	}

	return stat.IsDir()
}

// FileExists returns true if file exists and is a regular file, else false
func FileExists(fs afero.Fs, file string) bool {
	stat, err := fs.Stat(file)
	if err != nil {
		return false
	}

	return stat.Mode().IsRegular()
}

// EnsureDirExistsWithCorrectPermissions ensures the dir exists with the
// correct permissions
func EnsureDirExistsWithCorrectPermissions(fs afero.Fs, dir string, perms os.FileMode) error {
	if !dirExists(fs, dir) {
		log.Infof("Directory %s didn't exist so we're creating it", dir)
		if err := fs.MkdirAll(dir, perms); err != nil {
			return err
		}
	} else {
		// Ensure permissions of existing dir are correct
		if err := fs.Chmod(dir, perms); err != nil {
			return err
		}
	}

	return nil
}

// CopyFileForcefully copies src to dst, overwriting dst if it exists.
func CopyFileForcefully(fs afero.Fs, dst, src string, scriptPermissions os.FileMode) error {
	// Open dst
	dstFile, err := fs.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
		scriptPermissions)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// Open src (don't care about flags here)
	log.Info("Src file: ", src)
	srcFile, err := fs.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Copy it
	_, err = io.Copy(dstFile, srcFile)

	return err
}
