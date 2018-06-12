package sysuser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	testUID             = "12345678-1234-1234-1234-1234567890ab"
	testUsernameFromUID = "123456781234123412341234567890ab"
)

func TestUsernameFromContainershipUID(t *testing.T) {
	username := UsernameFromContainershipUID(testUID)
	assert.Equal(t, testUsernameFromUID, username, "")
}

func TestUsernameToContainershipUID(t *testing.T) {
	uid := UsernameToContainershipUID(testUsernameFromUID)
	assert.Equal(t, testUID, uid, "")
}
