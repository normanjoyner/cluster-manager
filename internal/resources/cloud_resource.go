package resources

import (
	"io/ioutil"
	"log"

	"github.com/containership/cloud-agent/internal/request"
	// containershipv3 "github.com/containership/cloud-agent/pkg/client/clientset/versioned/typed/containership.io/v3"
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

// Sync makes a request to cloud api for a resource and then writes the response
// to the resources cache
func Sync(cr CloudResource) error {
	bytes, err := makeRequest(cr.Endpoint())

	if err != nil {
		return err
	}

	if err := cr.UnmarshalToCache(bytes); err != nil {
		log.Printf("UnmarshalToCache failed for %s: %s\n", cr.Endpoint(), err.Error())
		return err
	}

	// TODO
	return nil
}

func makeRequest(endpoint string) ([]byte, error) {
	req, err := request.New(endpoint, "GET", nil)
	if err != nil {
		log.Printf("request.New failed: %s\n", err.Error())
		return nil, err
	}

	resp, err := req.MakeRequest()
	if err != nil {
		log.Printf("req.MakeRequest failed: %s\n", err.Error())
		return nil, err
	}
	defer resp.Body.Close()

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("ioutil.ReadAll failed: %s\n", err.Error())
		return nil, err
	}

	return bytes, nil
}
