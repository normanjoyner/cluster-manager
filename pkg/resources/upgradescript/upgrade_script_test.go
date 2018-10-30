package upgradescript

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	provisioncsv3 "github.com/containership/cluster-manager/pkg/apis/provision.containership.io/v3"
)

// TODO similar to authorized_keys_test.go, some strings are too coupled to the
// underlying fakes and hardcoded paths
const currentPath = "/etc/containership/scripts/upgrade/current"

const script = "#!/bin/bash\necho 'hallo'"
const upgradeType = provisioncsv3.UpgradeTypeKubernetes
const version = "v1.10.0"
const id = "12345678-1234-1234-1234-1234567890ab"

func TestWriteUpgradeScript(t *testing.T) {
	fs := afero.NewMemMapFs()
	scriptPath := GetUpgradeScriptFullPath(getUpgradeScriptFilename(upgradeType, version, id))
	writeUpgradeScript(fs, []byte(script), scriptPath)

	doesExist, err := afero.Exists(fs, scriptPath)
	assert.Nil(t, err)
	assert.True(t, doesExist)

	doesExist = exists(fs, upgradeType, version, id)
	assert.True(t, doesExist)
}

func TestWritePathToCurrent(t *testing.T) {
	fs := afero.NewMemMapFs()
	scriptPath := GetUpgradeScriptFullPath(getUpgradeScriptFilename(upgradeType, version, id))

	err := writePathToCurrent(fs, scriptPath)
	assert.Nil(t, err)

	exists, err := afero.Exists(fs, currentPath)
	assert.Nil(t, err)
	assert.True(t, exists)
}

func TestRemoveCurrent(t *testing.T) {
	fs := afero.NewMemMapFs()
	scriptPath := GetUpgradeScriptFullPath(getUpgradeScriptFilename(upgradeType, version, id))

	err := writePathToCurrent(fs, scriptPath)
	assert.Nil(t, err)

	exists, err := afero.Exists(fs, currentPath)
	assert.True(t, exists)

	err = removeCurrent(fs)
	exists, err = afero.Exists(fs, currentPath)
	assert.False(t, exists)
}
