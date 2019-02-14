package resources

import (
	"encoding/json"
	"fmt"

	cscloud "github.com/containership/csctl/cloud"

	csv3 "github.com/containership/cluster-manager/pkg/apis/containership.io/v3"
)

// CsUsers defines the Containership Cloud Users resource
type CsUsers struct {
	cloudResource
	cache []csv3.UserSpec
}

// NewCsUsers constructs a new CsUsers
func NewCsUsers(cloud cscloud.Interface) *CsUsers {
	cache := make([]csv3.UserSpec, 0)
	return &CsUsers{
		newCloudResource(cloud),
		cache,
	}
}

// Sync implements the CloudResource interface
func (us *CsUsers) Sync() error {
	users, err := us.cloud.API().Users(us.organizationID).List()
	if err != nil {
		return err
	}

	data, err := json.Marshal(users)
	if err != nil {
		return err
	}

	json.Unmarshal(data, &us.cache)

	return nil
}

// Cache return the containership users cache
func (us *CsUsers) Cache() []csv3.UserSpec {
	return us.cache
}

func sshKeyAreEqual(ukey csv3.SSHKeySpec, key csv3.SSHKeySpec) bool {
	return ukey.Name == key.Name &&
		ukey.Fingerprint == key.Fingerprint &&
		ukey.Key == key.Key
}

func sshKeysEqualCompare(specSSHKeys []csv3.SSHKeySpec, userSSHKeysByID map[string]csv3.SSHKeySpec) bool {
	if len(specSSHKeys) != len(userSSHKeysByID) {
		return false
	}

	specSSHKeysByID := make(map[string]csv3.SSHKeySpec, 0)
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
	spec, ok := specObj.(csv3.UserSpec)
	if !ok {
		return false, fmt.Errorf("The object is not of type UserSpec")
	}

	user, ok := parentSpecObj.(*csv3.User)
	if !ok {
		return false, fmt.Errorf("The object is not of type User")
	}

	if user.Spec.Name != spec.Name || user.Spec.AvatarURL != spec.AvatarURL {
		return false, nil
	}

	if len(user.Spec.SSHKeys) != len(spec.SSHKeys) {
		return false, nil
	}

	byID := make(map[string]csv3.SSHKeySpec, 0)
	for _, key := range user.Spec.SSHKeys {
		byID[key.ID] = key
	}

	return sshKeysEqualCompare(spec.SSHKeys, byID), nil
}
