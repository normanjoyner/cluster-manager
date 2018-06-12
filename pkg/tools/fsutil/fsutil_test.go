package fsutil

import (
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

// TODO better tests for this entire file, don't duplicate code

const filename = "/etc/containership/scripts/upgrade/current"
const permissions = os.FileMode(0600)

var data = []byte("test data")

func TestWriteNewFile(t *testing.T) {
	fs := afero.NewMemMapFs()

	err := WriteNewFile(fs, filename, data, os.FileMode(0600))
	assert.Nil(t, err)

	exists, err := afero.Exists(fs, filename)
	assert.Nil(t, err)
	assert.True(t, exists)

	stat, err := fs.Stat(filename)
	assert.Nil(t, err)
	assert.Equal(t, permissions.String(), stat.Mode().String())

	// Should fail if file already exists
	err = WriteNewFile(fs, filename, data, os.FileMode(0600))
	assert.Equal(t, os.ErrExist, err)
}

func TestWriteNewFileAtomic(t *testing.T) {
	fs := afero.NewMemMapFs()

	err := WriteNewFileAtomic(fs, filename, data, os.FileMode(0600))
	assert.Nil(t, err)

	exists, err := afero.Exists(fs, filename)
	assert.Nil(t, err)
	assert.True(t, exists)

	stat, err := fs.Stat(filename)
	assert.Nil(t, err)
	assert.Equal(t, permissions.String(), stat.Mode().String())

	// Should fail if file already exists
	err = WriteNewFileAtomic(fs, filename, data, os.FileMode(0600))
	assert.Equal(t, os.ErrExist, err)
}
