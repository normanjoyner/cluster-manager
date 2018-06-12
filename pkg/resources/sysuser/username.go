package sysuser

import (
	"fmt"
	"strings"
)

// UsernameFromContainershipUID returns the system username that the passed UID maps to
// TODO error handling needed? Probably.
func UsernameFromContainershipUID(uid string) string {
	return strings.Replace(uid, "-", "", -1)
}

// UsernameToContainershipUID returns the Containership UID that the passed system username maps to
func UsernameToContainershipUID(username string) string {
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		username[0:8],
		username[8:12],
		username[12:16],
		username[16:20],
		username[20:32])
}
