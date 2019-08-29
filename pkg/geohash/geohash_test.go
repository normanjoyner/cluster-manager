package geohash

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// These tests are naturally fragile given that the underlying data may change.
// If/when this becomes an issue, consider a new approach. Just sanity checking
// that nothing explodes for now.

func TestForProviderAndRegion(t *testing.T) {
	type testCase struct {
		provider string
		region   string

		shouldError bool
		expected    string

		message string
	}

	tests := []testCase{
		{
			provider:    "bad",
			region:      "us-east-1",
			shouldError: true,
			message:     "nonexistent provider",
		},
		{
			provider:    "amazon_web_services",
			region:      "us-east-1",
			shouldError: true,
			message:     "nonexistent provider due to errant spelling of aws provider",
		},
		{
			provider:    "aws",
			region:      "bad",
			shouldError: true,
			message:     "nonexistent region for aws",
		},
		{
			provider: "aws",
			region:   "us-east-1",
			expected: "dqbyhexqseyg",
			message:  "good entry for aws",
		},
		{
			provider:    "google",
			region:      "europe-west2",
			shouldError: true,
			message:     "nonexistent provider due to errant spelling of gce provider",
		},
		{
			provider:    "gce",
			region:      "bad",
			shouldError: true,
			message:     "nonexistent region for gce",
		},
		{
			provider: "gce",
			region:   "europe-west2",
			expected: "gcpvj0duq533",
			message:  "good entry for gce",
		},
	}

	for _, test := range tests {
		geohash, err := ForProviderAndRegion(test.provider, test.region)
		if test.shouldError {
			assert.Error(t, err, test.message)
		} else {
			assert.NoError(t, err, test.message)
			assert.Equal(t, test.expected, geohash, test.message)
		}
	}
}
