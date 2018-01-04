package v3test

import (
	"fmt"

	v3 "github.com/containership/cloud-agent/pkg/apis/containership.io/v3"
)

var userCounter int
var keyCounter int

const (
	// KeyString is the fake key string for all keys
	KeyString = "ssh-rsa ABCDEF"
	// FingerprintString is the fake fingerprint string for all keys
	FingerprintString = "00:11:22"
)

// uuidFromCounter generates a UUID from a counter
func uuidFromCounter(ctr int) string {
	return fmt.Sprintf("00000000-1111-2222-3333-%012d", ctr)
}

// nameFromCounter generates a name from a counter
func nameFromCounter(ctr int) string {
	return fmt.Sprintf("Name %012d", ctr)
}

// avatarURLFromCounter generates an avatar URL from a counter
func avatarURLFromCounter(ctr int) string {
	return fmt.Sprintf("http://avatar.url/%012d", ctr)
}

// timestampFromCounter generates a timestamp from a counter
func timestampFromCounter(ctr int) string {
	return string(ctr)
}

// generateUniqueKeys generates unique keys based off of a counter
func generateUniqueKeys(numKeys int) []v3.SSHKeySpec {
	keys := make([]v3.SSHKeySpec, numKeys)
	for i := 0; i < numKeys; i++ {
		keys[i] = v3.SSHKeySpec{
			ID:          uuidFromCounter(keyCounter),
			Name:        nameFromCounter(keyCounter),
			Fingerprint: FingerprintString,
			Key:         KeyString,
		}

		keyCounter++
	}

	return keys
}

// NewFakeUserSpec generates a new UserSpec, guaranteed to be
// unique thanks to a counter (ignoring overflow)
func NewFakeUserSpec(numKeys int) *v3.UserSpec {
	user := &v3.UserSpec{
		ID:        uuidFromCounter(userCounter),
		Name:      nameFromCounter(userCounter),
		AvatarURL: avatarURLFromCounter(userCounter),
		AddedAt:   timestampFromCounter(userCounter),
		SSHKeys:   generateUniqueKeys(numKeys),
	}

	userCounter++

	return user
}
