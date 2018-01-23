package buildinfo

import (
	"fmt"
)

// These variables should be initialized by ldflags
var (
	// GitDescribeString is the git describe for this build as set by ldflags
	GitDescribe string
	// GitCommitString is the git commit for this build as set by ldflags
	GitCommit string
)

// Spec represents the build info for the running binary
type Spec struct {
	// GitDescribe is the git describe for this build
	// It should be a clean git tag for a real release
	GitDescribe string `json:"git_describe"`
	// GitCommit is the git commit for this build
	GitCommit string `json:"git_commit"`
}

func init() {
	// If variables weren't initialized by ldflags, set them to
	// a dummy value
	if GitDescribe == "" {
		GitDescribe = "unknown"
	}

	if GitCommit == "" {
		GitCommit = "unknown"
	}
}

// Get returns a copy of the build info
func Get() Spec {
	return Spec{
		GitDescribe: GitDescribe,
		GitCommit:   GitCommit,
	}
}

// String returns a string representation of the build info
func String() string {
	return fmt.Sprintf("%s (%s)", GitDescribe, GitCommit)
}
