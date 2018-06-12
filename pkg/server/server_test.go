package server

import (
	"fmt"
	"testing"

	"github.com/containership/cloud-agent/pkg/k8sutil"

	"github.com/stretchr/testify/assert"

	"net/http"
	"net/http/httptest"
)

var a *CSServer

func TestMetadataGet(t *testing.T) {
	if k8sutil.API() == nil {
		fmt.Println("No configuration for kubernetes cluster, can't test server package")
		return
	}

	a = New()
	a.initialize()

	req, _ := http.NewRequest("GET", "/metadata", nil)
	response := executeRequest(req)

	checkResponseCode(t, http.StatusOK, response.Code)
}

func executeRequest(req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	a.Router.ServeHTTP(rr, req)

	return rr
}

func checkResponseCode(t *testing.T, expected, actual int) {
	assert.Equal(t, expected, actual)
}
