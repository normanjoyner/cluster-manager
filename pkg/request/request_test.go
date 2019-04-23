package request

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/containership/cluster-manager/pkg/env"
)

func TestNew(t *testing.T) {
	path := "/path"
	method := "GET"
	req, err := New(CloudServiceAPI, path, method, nil)
	assert.NoError(t, err)
	assert.NotNil(t, req)
}

func TestAppendToBaseURL(t *testing.T) {
	path := "/metadata"
	url := appendToBaseURL(CloudServiceAPI, path)
	expected := fmt.Sprintf("%s/v3/metadata", env.APIBaseURL())
	assert.Equal(t, expected, url)
}

func TestCreateClient(t *testing.T) {
	client := createClient()
	assert.NotNil(t, client)
}

func TestAddAuthHeader(t *testing.T) {
	path := "/path"
	method := "GET"
	req, err := New(CloudServiceAPI, path, method, nil)
	assert.NoError(t, err)
	assert.NotNil(t, req)

	req.addAuthHeader()

	authHeader := req.httpRequest.Header.Get("Authorization")

	// TODO mock env or os packages to make checking the token value here feasible
	assert.True(t, strings.HasPrefix(authHeader, "JWT "))
}

func TestAddContentTypeHeader(t *testing.T) {
	path := "/path"
	method := "GET"
	req, err := New(CloudServiceAPI, path, method, nil)
	assert.NoError(t, err)
	assert.NotNil(t, req)

	req.addContentTypeHeader()

	contentType := req.httpRequest.Header.Get("Content-Type")

	assert.Equal(t, "application/json", contentType)
}
