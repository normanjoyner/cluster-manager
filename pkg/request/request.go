package request

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httputil"
	"text/template"
	"time"

	"github.com/containership/cluster-manager/pkg/env"
	"github.com/containership/cluster-manager/pkg/log"
)

// Requester returns an object that can be used for making requests to the
// containership cloud api
type Requester struct {
	service     CloudService
	url         string
	method      string
	body        []byte
	httpRequest *http.Request
}

var urlParams = map[string]string{
	"OrganizationID": env.OrganizationID(),
	"ClusterID":      env.ClusterID(),
}

// New returns a Requester with the endpoint and type or request set that is
// needed to be made
func New(service CloudService, path, method string, body []byte) (*Requester, error) {
	p, err := getPath(path)
	if err != nil {
		return nil, err
	}

	url := appendToBaseURL(service, p)
	req, err := http.NewRequest(
		method,
		url,
		bytes.NewBuffer(body),
	)

	if err != nil {
		return nil, err
	}

	return &Requester{
		service:     service,
		httpRequest: req,
	}, nil
}

// AddHeader allows the user to add custom headers to the req property of a requester
func (r *Requester) AddHeader(key, value string) {
	r.httpRequest.Header.Set(key, value)
}

// Do makes a request using the req property of the requester
func (r *Requester) Do() (*http.Response, error) {
	client := createClient()

	return r.parseResponse(client.Do(r.httpRequest))
}

// MakeRequest builds a request that is able to speak with the Containership API
func (r *Requester) MakeRequest() (*http.Response, error) {
	r.addAuthHeader()

	if r.httpRequest.Body != nil {
		r.addContentTypeHeader()
	}

	client := createClient()

	return r.parseResponse(client.Do(r.httpRequest))
}

func (r *Requester) parseResponse(res *http.Response, err error) (*http.Response, error) {
	if err != nil {
		dumpRequest(r.httpRequest, true)
		return res, err
	}

	// The request succeeded, but the status code may be bad
	if res.StatusCode < http.StatusOK ||
		res.StatusCode >= http.StatusMultipleChoices {
		log.Debugf("%s responded with %d (%s)", r.service.String(), res.StatusCode,
			http.StatusText(res.StatusCode))

		// We can't dump the request body because it was already read
		dumpRequest(r.httpRequest, false)
		dumpResponse(res)

		return res, fmt.Errorf("Request returned with status code %d", res.StatusCode)
	}

	return res, nil
}

func (r *Requester) addAuthHeader() {
	r.AddHeader("Authorization", fmt.Sprintf("JWT %v", env.CloudClusterAPIKey()))
}

func (r *Requester) addContentTypeHeader() {
	// the content type for all cloud requests should be json
	r.AddHeader("Content-Type", "application/json")
}

func getPath(path string) (string, error) {
	tmpl, err := template.New("test").Parse(path)
	if err != nil {
		return "", err
	}

	var w bytes.Buffer
	err = tmpl.Execute(&w, urlParams)
	if err != nil {
		return "", err
	}

	return w.String(), nil
}

func appendToBaseURL(service CloudService, path string) string {
	base := service.BaseURL()
	return fmt.Sprintf("%s/v3%s", base, path)
}

func createClient() *http.Client {
	return &http.Client{
		Timeout: time.Second * 10,
	}
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
