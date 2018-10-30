package sysuser

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/containership/cluster-manager/pkg/apis/containership.io/v3"
	"github.com/containership/cluster-manager/pkg/apis/containership.io/v3/v3test"
)

// TODO the expected strings are too coupled to the underlying fakes and
// hardcoded paths

var testUserNoKeys = *v3test.NewFakeUserSpec(0)
var testUserNoKeysExpected = ""

var testUserOneKey = *v3test.NewFakeUserSpec(1)
var testUserOneKeyExpected = `command="/etc/containership/scripts/containership_login.sh 00000000111122223333000000000001" ssh-rsa ABCDEF
`

var testUserManyKeys = *v3test.NewFakeUserSpec(3)
var testUserManyKeysExpected = `command="/etc/containership/scripts/containership_login.sh 00000000111122223333000000000002" ssh-rsa ABCDEF
command="/etc/containership/scripts/containership_login.sh 00000000111122223333000000000002" ssh-rsa ABCDEF
command="/etc/containership/scripts/containership_login.sh 00000000111122223333000000000002" ssh-rsa ABCDEF
`

var allUsers = []v3.UserSpec{
	testUserNoKeys,
	testUserOneKey,
	testUserManyKeys,
	testUserNoKeys,
	testUserOneKey,
	testUserManyKeys,
}

var allUsersExpected = strings.Join([]string{
	testUserNoKeysExpected,
	testUserOneKeyExpected,
	testUserManyKeysExpected,
	testUserNoKeysExpected,
	testUserOneKeyExpected,
	testUserManyKeysExpected,
}, "")

func testBuildKeysStringNoKeys(t *testing.T) {
	s := buildKeysStringForUser(testUserNoKeys)
	assert.Equal(t, testUserNoKeysExpected, s)
}

func testBuildKeysStringOneKey(t *testing.T) {
	s := buildKeysStringForUser(testUserOneKey)
	assert.Equal(t, testUserOneKeyExpected, s)
}

func testBuildKeysStringManyKeys(t *testing.T) {
	s := buildKeysStringForUser(testUserManyKeys)
	assert.Equal(t, testUserManyKeysExpected, s)
}

func TestBuildKeysStringForUser(t *testing.T) {
	t.Run("NoKeys", testBuildKeysStringNoKeys)
	t.Run("OneKey", testBuildKeysStringOneKey)
	t.Run("ManyKeys", testBuildKeysStringManyKeys)
}

func TestBuildAllKeysString(t *testing.T) {
	s := buildAllKeysString(allUsers)
	assert.Equal(t, allUsersExpected, s)
}

func TestInitializeAuthorizedKeysFileStructure(t *testing.T) {
	fs := afero.NewMemMapFs()

	err := writeDummyContainerLoginScript(fs)
	assert.Nil(t, err)

	err = initializeAuthorizedKeysFileStructure(fs)
	assert.Nil(t, err)

	// Verify that all files and directories were created as expected
	exists, err := afero.DirExists(fs, getSSHDir())
	assert.Nil(t, err)
	assert.True(t, exists)

	exists, err = afero.Exists(fs, GetAuthorizedKeysFullPath())
	assert.Nil(t, err)
	assert.True(t, exists)

	exists, err = afero.Exists(fs, getLoginScriptFullPath())
	assert.Nil(t, err)
	assert.True(t, exists)

	verifyAllKnownPermissions(t, fs)
}

func TestWriteAuthorizedKeys(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Init is covered by a different test, assume it works
	err := writeDummyContainerLoginScript(fs)
	assert.Nil(t, err)
	err = initializeAuthorizedKeysFileStructure(fs)
	assert.Nil(t, err)

	filename := GetAuthorizedKeysFullPath()

	// Verify that we can write a bunch of keys properly
	err = writeAuthorizedKeys(fs, allUsers)
	assert.Nil(t, err)

	match, err := afero.FileContainsBytes(fs, filename,
		[]byte(allUsersExpected))
	assert.True(t, match)
	assert.Nil(t, err)

	// Verify that file contents are cleared if no keys users to write
	noUsers := make([]v3.UserSpec, 0)
	err = writeAuthorizedKeys(fs, noUsers)

	empty, err := afero.IsEmpty(fs, filename)
	assert.True(t, empty)
	assert.Nil(t, err)

	// Re-verify that we can write a bunch of keys properly after truncate
	err = writeAuthorizedKeys(fs, allUsers)
	assert.Nil(t, err)

	match, err = afero.FileContainsBytes(fs, filename,
		[]byte(allUsersExpected))
	assert.True(t, match)
	assert.Nil(t, err)

	// Mess up permissions and verify that a write fixes them
	fs.Chmod(getSSHDir(), os.ModeDir|os.FileMode(0777))
	fs.Chmod(filename, os.FileMode(0777))
	err = writeAuthorizedKeys(fs, allUsers)
	assert.Nil(t, err)
	verifyAllKnownPermissions(t, fs)

	// Verify that file contents are cleared if there are users but no keys to
	// write
	oneUserNoKeys := []v3.UserSpec{testUserNoKeys}
	err = writeAuthorizedKeys(fs, oneUserNoKeys)

	empty, err = afero.IsEmpty(fs, filename)
	assert.True(t, empty)
	assert.Nil(t, err)
}

func verifyAllKnownPermissions(t *testing.T, fs afero.Fs) {
	// TODO maybe add scripts stuff
	sshDir := getSSHDir()
	authKeysFile := GetAuthorizedKeysFullPath()

	dirInfo, err := fs.Stat(sshDir)
	assert.Nil(t, err)
	assert.Equal(t, sshDirPermissions.String(), dirInfo.Mode().String(),
		"SSH dir permissions bad")

	fileInfo, err := fs.Stat(authKeysFile)
	assert.Nil(t, err)
	assert.Equal(t, authorizedKeysPermissions.String(), fileInfo.Mode().String(),
		"authorized_keys permissions bad")
}

func writeDummyContainerLoginScript(fs afero.Fs) error {
	return afero.WriteFile(fs, loginScriptContainerPath,
		[]byte("#!/bin/bash\necho 'hallo'\n"), scriptPermissions)
}
