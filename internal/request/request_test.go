package request

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestMakeURL(t *testing.T) {
	path := "/metadata"
	url := makeURL(path)
	expected := "https://stage-api.containership.io/v2/organizations//clusters//metadata"

	if url != expected {
		t.Errorf("makeURL(%q) == %q, expected %q", path, url, expected)
	}
}

func TestCreateClient(t *testing.T) {
	client := createClient()
	expected := time.Second * 10

	if client.Timeout != expected {
		t.Errorf("createClient() timeout is %q but was expected to be %v", client.Timeout, expected)
	}
}

func TestAddHeaders(t *testing.T) {
	req, _ := http.NewRequest(
		"GET",
		"http://google.com",
		bytes.NewBuffer(make([]byte, 0)),
	)

	addHeaders(req)

	//TODO: update to JWT prefix
	if req.Header.Get("Authorization") != fmt.Sprintf("Bearer %v", os.Getenv("CONTAINERSHIP_CLOUD_CLUSTER_API_KEY")) {
		t.Errorf("addHeaders(req) Authorization header is %q but expected to be %q", req.Header.Get("Authorization"), fmt.Sprintf("Bearer %v", os.Getenv("CONTAINERSHIP_CLOUD_CLUSTER_API_KEY")))
	}
}

func TestMakeGetRequest(t *testing.T) {

}

func TestMakePostRequest(t *testing.T) {
}
