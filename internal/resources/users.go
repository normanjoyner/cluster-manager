package resources

import (
	"encoding/json"
	"fmt"

	containershipv3 "github.com/containership/cloud-agent/pkg/apis/containership.io/v3"
)

// CsUsers defines the Containership Cloud Users resource
type CsUsers struct {
	cloudResource
	cache []containershipv3.UserSpec
}

// NewCsUsers constructs a new CsUsers
func NewCsUsers() *CsUsers {
	return &CsUsers{
		cloudResource: cloudResource{
			endpoint: "/organizations/{{.OrganizationID}}/users",
		},
		cache: make([]containershipv3.UserSpec, 0),
	}
}

// UnmarshalToCache take the json returned from containership api
// and writes it to CsUsers cache
func (us *CsUsers) UnmarshalToCache(bytes []byte) error {
	return json.Unmarshal(bytes, &us.cache)
}

// Cache return the containership users cache
func (us *CsUsers) Cache() []containershipv3.UserSpec {
	return us.cache
}

func sshKeyAreEqual(ukey containershipv3.SSHKeySpec, key containershipv3.SSHKeySpec) bool {
	return ukey.Name == key.Name &&
		ukey.Fingerprint == key.Fingerprint &&
		ukey.Key == key.Key
}

func sshKeysEqualCompare(specSSHKeys []containershipv3.SSHKeySpec, userSSHKeysByID map[string]containershipv3.SSHKeySpec) bool {
	if len(specSSHKeys) != len(userSSHKeysByID) {
		return false
	}

	specSSHKeysByID := make(map[string]containershipv3.SSHKeySpec, 0)
	for _, key := range specSSHKeys {
		specSSHKeysByID[key.ID] = key
	}

	for id, key := range userSSHKeysByID {
		if !sshKeyAreEqual(key, specSSHKeysByID[id]) {
			return false
		}
	}

	return true
}

// IsEqual compares a UserSpec to another User
func (us *CsUsers) IsEqual(specObj interface{}, parentSpecObj interface{}) (bool, error) {
	spec, ok := specObj.(containershipv3.UserSpec)
	if !ok {
		return false, fmt.Errorf("The object is not of type UserSpec")
	}

	user, ok := parentSpecObj.(*containershipv3.User)
	if !ok {
		return false, fmt.Errorf("The object is not of type User")
	}

	if user.Spec.Name != spec.Name || user.Spec.AvatarURL != spec.AvatarURL {
		return false, nil
	}

	if len(user.Spec.SSHKeys) != len(spec.SSHKeys) {
		return false, nil
	}

	byID := make(map[string]containershipv3.SSHKeySpec, 0)
	for _, key := range user.Spec.SSHKeys {
		byID[key.ID] = key
	}

	return sshKeysEqualCompare(spec.SSHKeys, byID), nil
}
