package buildinfo

import (
	"fmt"
)

// These variables should be initialized by ldflags
var (
	// GitDescribe is the git describe for this build
	GitDescribe string
	// GitCommit is the git commit for this build
	GitCommit string
)

// String returns a string representation of the build info
func String() string {
	describe := GitDescribe
	if describe == "" {
		describe = "<unknown git describe>"
	}

	commit := GitDescribe
	if commit == "" {
		commit = "<unknown git commit>"
	}

	return fmt.Sprintf("%s (%s)", describe, commit)
}
