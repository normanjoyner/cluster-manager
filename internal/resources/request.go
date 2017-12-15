package resources

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/containership/cloud-agent/internal/envvars"
)

func makeURL(path string) string {
	// TODO: update to v3 API
	return fmt.Sprintf("%s/v2/organizations/%s/clusters/%s%s", envvars.GetBaseURL(), envvars.GetOrganizationID(), envvars.GetClusterID(), path)
}

func addHeaders(req *http.Request) {
	// TODO: update to JWT prefix
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", envvars.GetCloudClusterAPIKey()))
}

func createClient() *http.Client {
	return &http.Client{
		Timeout: time.Second * 10,
	}
}

// MakeRequest builds a request that is able to speak with the Containership API
func MakeRequest(path, method string, jsonBody []byte) (*http.Response, error) {
	url := makeURL(path)
	req, err := http.NewRequest(
		method,
		url,
		bytes.NewBuffer(jsonBody),
	)

	if err == nil {
		log.Printf("Request %v \n", req)
	}

	addHeaders(req)
	client := createClient()

	res, err := client.Do(req)

	if err != nil {
		return res, err
	}

	return res, nil
}
