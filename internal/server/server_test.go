package server

import (
	"fmt"
	"testing"
	"os"

	"github.com/stretchr/testify/assert"

	"net/http"
	"net/http/httptest"
)

var a CSServer

func TestMetadataGet(t *testing.T) {
		fmt.Println("hereeeee")
	if os.Getenv("KUBERNETES_SERVICE_HOST") == "" {
		fmt.Println("No configuration for kubernetes cluster, can't test server package")
		return
	}

	a = CSServer{}
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
