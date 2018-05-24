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
	service CloudService
	url     string
	method  string
	body    []byte
}

var urlParams = map[string]string{
	"OrganizationID": env.OrganizationID(),
	"ClusterID":      env.ClusterID(),
	"NodeName":       env.NodeName(),
}

// New returns a Requester with the endpoint and type or request set that is
// needed to be made
func New(service CloudService, path, method string, body []byte) (*Requester, error) {
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
		service: service,
		url:     appendToBaseURL(service, p),
		method:  method,
		body:    body,
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

func appendToBaseURL(service CloudService, path string) string {
	var base string
	switch service {
	case CloudServiceAPI:
		base = env.APIBaseURL()
	case CloudServiceProvision:
		base = env.ProvisionBaseURL()
	}

	return fmt.Sprintf("%s/v3%s", base, path)
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
		log.Debugf("Failed request: %+v", *req)
		return res, err
	}

	// Log the error code and request here and we'll log the response body
	// in Unmarshal
	if res.StatusCode < http.StatusOK ||
		res.StatusCode >= http.StatusMultipleChoices {
		log.Debugf("%s responded with %d (%s)", r.service.String(), res.StatusCode,
			http.StatusText(res.StatusCode))
		log.Debugf("Request: %+v", *req)

		return res, fmt.Errorf("Request returned with status code %d", res.StatusCode)
	}

	return res, nil
}
