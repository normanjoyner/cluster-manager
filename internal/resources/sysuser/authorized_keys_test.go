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

	err := writeAuthorizedKeys(fs, allUsers)
	assert.Nil(t, err)

	filename := buildAuthorizedKeysFullPath()

	match, err := afero.FileContainsBytes(fs, filename,
		[]byte(allUsersExpected))

	assert.True(t, match)
	assert.Nil(t, err)
}
