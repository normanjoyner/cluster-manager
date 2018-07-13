package fsutil

import (
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/afero"

	"github.com/containership/cloud-agent/pkg/log"
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

// IsEmpty returns true if the directory passed in contains no files
func IsEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err // Either not empty or error, suits both cases
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
func CopyFileForcefully(fs afero.Fs, dst, src string, perms os.FileMode) error {
	// Open dst
	dstFile, err := fs.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perms)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// Open src (don't care about flags here)
	srcFile, err := fs.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Copy it
	_, err = io.Copy(dstFile, srcFile)

	return err
}

// WriteNewFile writes data to the file specified by filename.
// If the file already exists, then an error will be returned.
func WriteNewFile(fs afero.Fs, filename string, data []byte, perms os.FileMode) error {
	// TODO O_EXCL is not yet supported for the afero memmap fs, so we need this
	// workaround. See https://github.com/spf13/afero/pull/102.
	if FileExists(fs, filename) {
		return os.ErrExist
	}

	// O_EXCL is used to force failure if the file already exists so it doesn't
	// get overwritten
	f, err := fs.OpenFile(filename, os.O_CREATE|os.O_EXCL|os.O_WRONLY, perms)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(data)
	if err != nil {
		return err
	}

	return nil
}

// WriteNewFileAtomic writes data first to a temporary file (in the same directory
// as the desired file) and then moves it to filename. To a consumer of the
// file located at filename, it appears that the file was created with the
// given data atomically (rather than being opened/created and subsequently
// written to).
// Like WriteNewFile, this function will error out if the destination file
// already exists.
func WriteNewFileAtomic(fs afero.Fs, filename string, data []byte, perms os.FileMode) error {
	if FileExists(fs, filename) {
		return os.ErrExist
	}

	dir, tmpPrefix := filepath.Split(filename)
	tmpFile, err := afero.TempFile(fs, dir, tmpPrefix)
	if err != nil {
		return err
	}
	defer tmpFile.Close()

	if err := fs.Chmod(tmpFile.Name(), perms); err != nil {
		return err
	}

	if _, err := tmpFile.Write(data); err != nil {
		return err
	}

	return fs.Rename(tmpFile.Name(), filename)
}
