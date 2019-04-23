package resources

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	csv3 "github.com/containership/cluster-manager/pkg/apis/containership.io/v3"
)

var key1spec = csv3.SSHKeySpec{
	ID:          "1",
	Name:        "key1",
	Fingerprint: "key1fingerprint",
	Key:         "keykey1",
}
var key1specb = csv3.SSHKeySpec{
	ID:          "1",
	Name:        "key1",
	Fingerprint: "key1fingerprint",
	Key:         "keykey1",
}
var key2spec = csv3.SSHKeySpec{
	ID:          "2",
	Name:        "key2",
	Fingerprint: "key2fingerprint",
	Key:         "keykey2",
}
var key3spec = csv3.SSHKeySpec{
	ID:          "3",
	Name:        "key3",
	Fingerprint: "key3fingerprint",
	Key:         "keykey3",
}

func TestSSHKeyAreEqual(t *testing.T) {
	// check for both being empty
	emptySameTest := sshKeyAreEqual(csv3.SSHKeySpec{}, csv3.SSHKeySpec{})
	assert.Equal(t, emptySameTest, true)

	// check one spec being empty
	emptyDiffTest := sshKeyAreEqual(key1spec, csv3.SSHKeySpec{})
	assert.Equal(t, emptyDiffTest, false)
	emptyDiffTest2 := sshKeyAreEqual(csv3.SSHKeySpec{}, key1spec)
	assert.Equal(t, emptyDiffTest2, false)

	// check keys with different data
	differentTest := sshKeyAreEqual(key1spec, key2spec)
	assert.Equal(t, differentTest, false)

	// make sure it's checking data not address
	sameTest := sshKeyAreEqual(key1spec, key1specb)
	assert.Equal(t, sameTest, true)
}

func createMap(k ...csv3.SSHKeySpec) map[string]csv3.SSHKeySpec {
	value := make(map[string]csv3.SSHKeySpec, 0)
	for _, v := range k {
		value[v.ID] = v
	}

	return value
}

//sshKeysEqualCompare(specSSHKeys []csv3.SSHKeySpec, userSSHKeysByID map[string]csv3.SSHKeySpec)
func TestSSHKeysEqualCompare(t *testing.T) {
	// check for both being empty
	emptySameTest := sshKeysEqualCompare([]csv3.SSHKeySpec{}, make(map[string]csv3.SSHKeySpec, 0))
	assert.Equal(t, true, emptySameTest)

	// check map being empty
	emptyDiffTest := sshKeysEqualCompare([]csv3.SSHKeySpec{key1spec, key2spec}, make(map[string]csv3.SSHKeySpec, 0))
	assert.Equal(t, false, emptyDiffTest)
	// check spec being empty
	mapOfKeys := createMap(key1spec, key2spec)
	emptyDiffTest2 := sshKeysEqualCompare([]csv3.SSHKeySpec{}, mapOfKeys)
	assert.Equal(t, false, emptyDiffTest2)

	// check keys with different data same length
	differentTest := sshKeysEqualCompare([]csv3.SSHKeySpec{key1spec, key3spec}, mapOfKeys)
	assert.Equal(t, false, differentTest)
	// check keys with different data different length
	differentTest2 := sshKeysEqualCompare([]csv3.SSHKeySpec{key1spec, key1spec, key3spec}, mapOfKeys)
	assert.Equal(t, false, differentTest2)
	differentTest3 := sshKeysEqualCompare([]csv3.SSHKeySpec{key1spec}, mapOfKeys)
	assert.Equal(t, false, differentTest3)

	// make sure it's checking data not address
	sameTest := sshKeysEqualCompare([]csv3.SSHKeySpec{key1spec, key2spec}, mapOfKeys)
	assert.Equal(t, true, sameTest, false)
}

var emptyUserSpec = csv3.UserSpec{}
var emptyUser = &csv3.User{}

var user1 = &csv3.User{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "user1",
		Namespace: "containership",
	},
	Spec: csv3.UserSpec{
		ID:        "1",
		Name:      "user1",
		AvatarURL: "https://www.gravatar.com/avatar",
		AddedAt:   "123",
		SSHKeys:   []csv3.SSHKeySpec{key1spec, key2spec},
	},
}

func TestIsEqual(t *testing.T) {
	c := NewCsUsers(nil)

	_, err := c.IsEqual("wrong type", emptyUser)
	assert.Error(t, err, "bad spec type")

	_, err = c.IsEqual(emptyUserSpec, "wrong type")
	assert.Error(t, err, "bad parent type")

	eq, err := c.IsEqual(emptyUserSpec, emptyUser)
	assert.NoError(t, err)
	assert.True(t, eq, "both empty")

	eq, err = c.IsEqual(emptyUserSpec, user1)
	assert.NoError(t, err)
	assert.False(t, eq, "spec only empty")

	eq, err = c.IsEqual(user1.Spec, emptyUser)
	assert.NoError(t, err)
	assert.False(t, eq, "parent only empty")

	same := user1.DeepCopy().Spec
	eq, err = c.IsEqual(same, user1)
	assert.NoError(t, err)
	assert.True(t, eq, "copied spec")

	diff := user1.DeepCopy().Spec
	diff.ID = "different"
	eq, err = c.IsEqual(diff, user1)
	assert.NoError(t, err)
	assert.False(t, eq, "different ID")

	diff = user1.DeepCopy().Spec
	diff.AddedAt = "different"
	eq, err = c.IsEqual(diff, user1)
	assert.NoError(t, err)
	assert.False(t, eq, "different added_at")

	diff = user1.DeepCopy().Spec
	diff.AvatarURL = "different"
	eq, err = c.IsEqual(diff, user1)
	assert.NoError(t, err)
	assert.False(t, eq, "different avatar URL")

	diff = user1.DeepCopy().Spec
	diff.SSHKeys = nil
	eq, err = c.IsEqual(diff, user1)
	assert.NoError(t, err)
	assert.False(t, eq, "different keys - one nil")

	diff = user1.DeepCopy().Spec
	diff.SSHKeys = []csv3.SSHKeySpec{}
	eq, err = c.IsEqual(diff, user1)
	assert.NoError(t, err)
	assert.False(t, eq, "different keys - one empty")

	// Remaining SSH key equal tests are covered elsewhere
}

func TestUsersCache(t *testing.T) {
	userBytes := []byte(`[{
	"id" : "1234",
	"name" : "name",
	"avatar_url" : "https://testing.com",
	"added_at" : "timestring",
	"ssh_keys" : [{
		"id" : "2345",
		"name" : "ssh key",
		"fingerprint" : "fingerprint",
		"key" : "key"
	}]
}]`)

	u := NewCsUsers(nil)

	err := json.Unmarshal(userBytes, &u.cache)
	assert.NoError(t, err, "unmarshal good data")

	c := u.Cache()
	assert.Equal(t, u.cache, c)
}
