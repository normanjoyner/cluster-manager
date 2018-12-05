package resources

import (
	"io/ioutil"

	"github.com/containership/cluster-manager/pkg/log"
	"github.com/containership/cluster-manager/pkg/request"
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
	// Service returns the request.CloudService type of the API to make a request to
	Service() request.CloudService
}

// cloudResource defines what each resource needs to contain
type cloudResource struct {
	endpoint string
	service  request.CloudService
}

func (cr cloudResource) Endpoint() string {
	return cr.endpoint
}

func (cr cloudResource) Service() request.CloudService {
	return cr.service
}

// Sync makes a request to cloud api for a resource and then writes the response
// to the resources cache
func Sync(cr CloudResource) error {
	bytes, err := makeRequest(cr.Service(), cr.Endpoint())
	if err != nil {
		return err
	}

	err = cr.UnmarshalToCache(bytes)
	if err != nil {
		log.Debugf("Bad response: %s", string(bytes))
	}
	return err
}

func makeRequest(service request.CloudService, endpoint string) ([]byte, error) {
	req, err := request.New(service, endpoint, "GET", nil)
	if err != nil {
		return nil, err
	}

	resp, err := req.MakeRequest()
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}
