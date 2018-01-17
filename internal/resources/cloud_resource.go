package resources

import (
	"io/ioutil"
	"net/http"

	"github.com/containership/cloud-agent/internal/log"
	"github.com/containership/cloud-agent/internal/request"
)

// CloudResource defines an interface for resources to adhere to in order to be kept
// in sync with Containership Cloud
type CloudResource interface {
	// Endpoint returns the API endpoint associated with this cloud resource
	Endpoint() string
	// UnmarshalToCache unmarshals to the resource's underlying cache
	// TODO use stream not []byte for efficiency
	UnmarshalToCache(bytes []byte) error
	// IsEqual compares a spec to it's parent object spec
	IsEqual(spec interface{}, parentSpecObj interface{}) (bool, error)
}

// cloudResource defines what each resource needs to contain
type cloudResource struct {
	endpoint string
}

func (cr cloudResource) Endpoint() string {
	return cr.endpoint
}

// Sync makes a request to cloud api for a resource and then writes the response
// to the resources cache
func Sync(cr CloudResource) error {
	bytes, err := makeRequest(cr.Endpoint())
	if err != nil {
		return err
	}

	err = cr.UnmarshalToCache(bytes)
	if err != nil {
		log.Debugf("Bad response: %s\n", string(bytes))
	}
	return err
}

func makeRequest(endpoint string) ([]byte, error) {
	req, err := request.New(endpoint, "GET", nil)
	if err != nil {
		return nil, err
	}

	resp, err := req.MakeRequest()
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Log the error code and request here and we'll log the response body
	// in Unmarshal
	if resp.StatusCode < http.StatusOK ||
		resp.StatusCode >= http.StatusMultipleChoices {
		log.Debugf("Cloud API responded with %d (%s)\n", resp.StatusCode,
			http.StatusText(resp.StatusCode))
		log.Debugf("Request: %+v\n", *req)
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}
