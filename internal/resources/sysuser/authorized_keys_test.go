package sysuser

import (
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/containership/cloud-agent/pkg/apis/containership.io/v3"
	"github.com/containership/cloud-agent/pkg/apis/containership.io/v3/v3test"
)

// TODO the expected strings are too coupled to the underlying fakes and
// hardcoded paths

var testUserNoKeys = *v3test.NewFakeUserSpec(0)
var testUserNoKeysExpected = ""

var testUserOneKey = *v3test.NewFakeUserSpec(1)
var testUserOneKeyExpected = `command="/opt/containership/home/containership_login.sh 00000000111122223333000000000001" ssh-rsa ABCDEF
`

var testUserManyKeys = *v3test.NewFakeUserSpec(3)
var testUserManyKeysExpected = `command="/opt/containership/home/containership_login.sh 00000000111122223333000000000002" ssh-rsa ABCDEF
command="/opt/containership/home/containership_login.sh 00000000111122223333000000000002" ssh-rsa ABCDEF
command="/opt/containership/home/containership_login.sh 00000000111122223333000000000002" ssh-rsa ABCDEF
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

func TestWriteAuthorizedKeys(t *testing.T) {
	fs := afero.NewMemMapFs()

	filename := buildAuthorizedKeysFullPath()

	// New FS so file should not exist yet
	exists, err := afero.Exists(fs, filename)
	assert.False(t, exists)
	assert.Nil(t, err)

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

	// Verify that file contents are cleared if there are users but no keys to
	// write
	oneUserNoKeys := []v3.UserSpec{testUserNoKeys}
	err = writeAuthorizedKeys(fs, oneUserNoKeys)

	empty, err = afero.IsEmpty(fs, filename)
	assert.True(t, empty)
	assert.Nil(t, err)
}
