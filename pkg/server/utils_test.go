package server

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type tokenTest struct {
	input    string
	expected expectedResponse
	message  string
}

type expectedResponse struct {
	err error
	s   string
}

var tests = []tokenTest{
	{
		input: "",
		expected: expectedResponse{
			err: fmt.Errorf("Authorization header is empty"),
			s:   "",
		},
		message: "Expected error for empty Auth Header",
	},
	{
		input: "JWT",
		expected: expectedResponse{
			err: fmt.Errorf("JWT token is ill formated"),
			s:   "",
		},
		message: "Expect error for JWT token being empty",
	},
	{
		input: "JWT eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
		expected: expectedResponse{
			err: nil,
			s:   "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
		},
		message: "Get auth token for correctly formated auth header",
	},
}

func TestGetJWT(t *testing.T) {
	for _, test := range tests {
		result, err := getJWT(test.input)
		// Note that this is a deep compare
		assert.Equal(t, test.expected.err, err, test.message)
		assert.Equal(t, test.expected.s, result, test.message)
	}
}
