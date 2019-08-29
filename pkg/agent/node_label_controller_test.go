package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProviderNameFromProviderID(t *testing.T) {
	type testCase struct {
		providerID string
		expected   string
		message    string
	}

	tests := []testCase{
		{
			providerID: "digitalocean://abcd",
			expected:   "digitalocean",
			message:    "good provider ID",
		},
		{
			providerID: "amazon_web_services://abcd",
			expected:   "amazon_web_services",
			message:    "good provider ID with underscores",
		},
		{
			providerID: "://abcd",
			expected:   "",
			message:    "empty provider name",
		},
		{
			providerID: "://",
			expected:   "",
			message:    "empty provider name and node ID",
		},
		{
			providerID: "",
			expected:   "",
			message:    "empty string",
		},
		{
			providerID: "malformed",
			expected:   "",
			message:    "malformed",
		},
	}

	for _, test := range tests {
		providerName := providerNameFromProviderID(test.providerID)
		assert.Equal(t, test.expected, providerName, test.message)
	}
}
