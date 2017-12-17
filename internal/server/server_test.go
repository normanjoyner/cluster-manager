package server

import (
	"testing"

	"net/http"
	"net/http/httptest"
)

var a CSServer

func TestMetadataGet(t *testing.T) {
	a = CSServer{}
	a.initialize()

	req, _ := http.NewRequest("GET", "/metadata", nil)
	response := executeRequest(req)

	checkResponseCode(t, http.StatusInternalServerError, response.Code)
}

func executeRequest(req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	a.Router.ServeHTTP(rr, req)

	return rr
}

func checkResponseCode(t *testing.T, expected, actual int) {
	if expected != actual {
		t.Errorf("Expected response code %d. Got %d\n", expected, actual)
	}
}
