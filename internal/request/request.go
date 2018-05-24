package request

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httputil"
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
		dumpRequest(req, true)
		return res, err
	}

	// The request succeeded, but the status code may be bad
	if res.StatusCode < http.StatusOK ||
		res.StatusCode >= http.StatusMultipleChoices {
		log.Debugf("%s responded with %d (%s)", r.service.String(), res.StatusCode,
			http.StatusText(res.StatusCode))

		// We can't dump the request body because it was already read
		dumpRequest(req, false)
		dumpResponse(res)

		return res, fmt.Errorf("Request returned with status code %d", res.StatusCode)
	}

	return res, nil
}

// dumpRequest attempts to dump an HTTP request for debug purposes
func dumpRequest(req *http.Request, dumpBody bool) {
	dump, err := httputil.DumpRequestOut(req, dumpBody)
	if err != nil {
		dump = []byte(fmt.Sprintf("Error dumping request: %s", err))
	}
	log.Debugf("Request: %q", string(dump))
}

// dumpRequest attempts to dump an HTTP response for debug purposes
func dumpResponse(res *http.Response) {
	dump, err := httputil.DumpResponse(res, true)
	if err != nil {
		dump = []byte(fmt.Sprintf("Error dumping response: %s", err))
	}
	log.Debugf("Response: %q", string(dump))
}
