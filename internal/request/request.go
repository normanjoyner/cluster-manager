package request

import (
	"bytes"
	"fmt"
	"net/http"
	"text/template"
	"time"

	"github.com/containership/cloud-agent/internal/env"
	"github.com/containership/cloud-agent/internal/log"
)

// Requester returns an object that can be used for making requests to the
// containership cloud api
type Requester struct {
	url    string
	method string
	body   []byte
}

var urlParams = map[string]string{
	"OrganizationID": env.OrganizationID(),
	"ClusterID":      env.ClusterID(),
}

// New returns a Requester with the endpoint and type or request set that is
// needed to be made
func New(path, method string, body []byte) (*Requester, error) {
	tmpl, err := template.New("test").Parse(path)

	if err != nil {
		return nil, err
	}

	var w bytes.Buffer
	err = tmpl.Execute(&w, urlParams)

	if err != nil {
		return nil, err
	}

	p := w.String()

	return &Requester{
		url:    appendToBaseURL(p),
		method: method,
		body:   body,
	}, nil
}

// URL returns the url that has been set for requests
func (r *Requester) URL() string {
	return r.url
}

// Method returns the method that has been set for request
func (r *Requester) Method() string {
	return r.method
}

// Body returns the current body set for a request
func (r *Requester) Body() []byte {
	return r.body
}

func appendToBaseURL(path string) string {
	return fmt.Sprintf("%s/v3%s", env.BaseURL(), path)
}

func addHeaders(req *http.Request) {
	req.Header.Set("Authorization", fmt.Sprintf("JWT %v", env.CloudClusterAPIKey()))
}

func createClient() *http.Client {
	return &http.Client{
		Timeout: time.Second * 10,
	}
}

// MakeRequest builds a request that is able to speak with the Containership API
func (r *Requester) MakeRequest() (*http.Response, error) {
	req, err := http.NewRequest(
		r.method,
		r.url,
		bytes.NewBuffer(r.body),
	)
	addHeaders(req)

	client := createClient()

	res, err := client.Do(req)
	if err != nil {
		log.Debugf("Failed request: %+v\n", *req)
		return res, err
	}

	// Log the error code and request here and we'll log the response body
	// in Unmarshal
	if res.StatusCode < http.StatusOK ||
		res.StatusCode >= http.StatusMultipleChoices {
		log.Debugf("Cloud API responded with %d (%s)\n", res.StatusCode,
			http.StatusText(res.StatusCode))
		log.Debugf("Request: %+v\n", *req)

		return res, fmt.Errorf("Request returned with status code %d", res.StatusCode)
	}

	return res, nil
}
