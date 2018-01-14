package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"

	containershipv3 "github.com/containership/cloud-agent/pkg/apis/containership.io/v3"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var key1spec = containershipv3.SSHKeySpec{
	ID:          "1",
	Name:        "key1",
	Fingerprint: "key1fingerprint",
	Key:         "keykey1",
}
var key1specb = containershipv3.SSHKeySpec{
	ID:          "1",
	Name:        "key1",
	Fingerprint: "key1fingerprint",
	Key:         "keykey1",
}
var key2spec = containershipv3.SSHKeySpec{
	ID:          "2",
	Name:        "key2",
	Fingerprint: "key2fingerprint",
	Key:         "keykey2",
}
var key3spec = containershipv3.SSHKeySpec{
	ID:          "3",
	Name:        "key3",
	Fingerprint: "key3fingerprint",
	Key:         "keykey3",
}

func TestSSHKeyAreEqual(t *testing.T) {
	// check for both being empty
	emptySameTest := sshKeyAreEqual(containershipv3.SSHKeySpec{}, containershipv3.SSHKeySpec{})
	assert.Equal(t, emptySameTest, true)

	// check one spec being empty
	emptyDiffTest := sshKeyAreEqual(key1spec, containershipv3.SSHKeySpec{})
	assert.Equal(t, emptyDiffTest, false)
	emptyDiffTest2 := sshKeyAreEqual(containershipv3.SSHKeySpec{}, key1spec)
	assert.Equal(t, emptyDiffTest2, false)

	// check keys with different data
	differentTest := sshKeyAreEqual(key1spec, key2spec)
	assert.Equal(t, differentTest, false)

	// make sure it's checking data not address
	sameTest := sshKeyAreEqual(key1spec, key1specb)
	assert.Equal(t, sameTest, true)
}

func createMap(k ...containershipv3.SSHKeySpec) map[string]containershipv3.SSHKeySpec {
	value := make(map[string]containershipv3.SSHKeySpec, 0)
	for _, v := range k {
		value[v.ID] = v
	}

	return value
}

//sshKeysEqualCompare(specSSHKeys []containershipv3.SSHKeySpec, userSSHKeysByID map[string]containershipv3.SSHKeySpec)
func TestSSHKeysEqualCompare(t *testing.T) {
	// check for both being empty
	emptySameTest := sshKeysEqualCompare([]containershipv3.SSHKeySpec{}, make(map[string]containershipv3.SSHKeySpec, 0))
	assert.Equal(t, true, emptySameTest)

	// check map being empty
	emptyDiffTest := sshKeysEqualCompare([]containershipv3.SSHKeySpec{key1spec, key2spec}, make(map[string]containershipv3.SSHKeySpec, 0))
	assert.Equal(t, false, emptyDiffTest)
	// check spec being empty
	mapOfKeys := createMap(key1spec, key2spec)
	emptyDiffTest2 := sshKeysEqualCompare([]containershipv3.SSHKeySpec{}, mapOfKeys)
	assert.Equal(t, false, emptyDiffTest2)

	// check keys with different data same length
	differentTest := sshKeysEqualCompare([]containershipv3.SSHKeySpec{key1spec, key3spec}, mapOfKeys)
	assert.Equal(t, false, differentTest)
	// check keys with different data different length
	differentTest2 := sshKeysEqualCompare([]containershipv3.SSHKeySpec{key1spec, key1spec, key3spec}, mapOfKeys)
	assert.Equal(t, false, differentTest2)
	differentTest3 := sshKeysEqualCompare([]containershipv3.SSHKeySpec{key1spec}, mapOfKeys)
	assert.Equal(t, false, differentTest3)

	// make sure it's checking data not address
	sameTest := sshKeysEqualCompare([]containershipv3.SSHKeySpec{key1spec, key2spec}, mapOfKeys)
	assert.Equal(t, true, sameTest, false)
}

var emptyUser = &containershipv3.User{}

var user1 = &containershipv3.User{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "user1",
		Namespace: "containership",
	},
	Spec: containershipv3.UserSpec{
		ID:      "1",
		Name:    "User1",
		SSHKeys: []containershipv3.SSHKeySpec{key1spec},
	},
}

var user1spec = containershipv3.UserSpec{
	ID:      "1",
	Name:    "User1",
	SSHKeys: []containershipv3.SSHKeySpec{key1spec},
}

var user2 = &containershipv3.User{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "user2",
		Namespace: "containership",
	},
	Spec: containershipv3.UserSpec{
		ID:      "2",
		Name:    "User2",
		SSHKeys: []containershipv3.SSHKeySpec{key1spec, key2spec},
	},
}

var user2specDiff = containershipv3.UserSpec{
	ID:      "2",
	Name:    "User2",
	SSHKeys: []containershipv3.SSHKeySpec{key3spec, key2spec},
}

func TestIsEqual(t *testing.T) {
	c := NewCsUsers()
	// check for both being empty
	emptySameTest, err := c.IsEqual(containershipv3.UserSpec{}, emptyUser)
	assert.Nil(t, err)
	assert.Equal(t, emptySameTest, true)

	// check with spec being empty, and user being empty
	emptyDiffTest, err := c.IsEqual(user1spec, emptyUser)
	assert.Nil(t, err)
	assert.Equal(t, emptyDiffTest, false)
	emptyDiffTest2, err := c.IsEqual(containershipv3.UserSpec{}, user1)
	assert.Nil(t, err)
	assert.Equal(t, emptyDiffTest2, false)

	// assert they are the same
	sameTest, err := c.IsEqual(user1spec, user1)
	assert.Nil(t, err)
	assert.Equal(t, sameTest, true)

	// check same with different keys same length
	differentTest, err := c.IsEqual(user2specDiff, user2)
	assert.Nil(t, err)
	assert.Equal(t, differentTest, false)

	// check with different keys, different lengths
	diffLengths, err := c.IsEqual(user2specDiff, user1)
	assert.Nil(t, err)
	assert.Equal(t, diffLengths, false)
}
