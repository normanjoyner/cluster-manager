package request

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"text/template"
	"time"

	"github.com/containership/cloud-agent/internal/envvars"
)

type Requester struct {
	url    string
	method string
	body   []byte
}

var urlParams = map[string]string{
	"OrganizationID": envvars.GetOrganizationID(),
	"ClusterID":      envvars.GetClusterID(),
}

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

func (r *Requester) URL() string {
	return r.url
}

func (r *Requester) Method() string {
	return r.method
}

func (r *Requester) Body() []byte {
	return r.body
}

func appendToBaseURL(path string) string {
	// TODO: update to v3 API
	return fmt.Sprintf("%s/v2%s", envvars.GetBaseURL(), path)
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
func (r *Requester) MakeRequest() (*http.Response, error) {
	req, err := http.NewRequest(
		r.method,
		r.url,
		bytes.NewBuffer(r.body),
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
