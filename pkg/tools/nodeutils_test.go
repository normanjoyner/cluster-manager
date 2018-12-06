package tools

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetImageVersion(t *testing.T) {
	imageVersion := "v1.12.1"
	image := "containership/test-image"
	urlWithPort := "registry.com:8080"

	version := getImageVersion(image)
	assert.Equal(t, "latest", version)

	version = getImageVersion(fmt.Sprintf("%s:%s", image, imageVersion))
	assert.Equal(t, imageVersion, version)

	version = getImageVersion(fmt.Sprintf("%s/%s:%s", urlWithPort, image, imageVersion))
	assert.Equal(t, imageVersion, version)
}
