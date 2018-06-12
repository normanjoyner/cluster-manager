package k8sutil

import (
	"time"

	"k8s.io/client-go/rest"

	csclientset "github.com/containership/cloud-agent/pkg/client/clientset/versioned"
	csinformers "github.com/containership/cloud-agent/pkg/client/informers/externalversions"
)

// CSKubeAPI is an object for interacting with the CRDs containership
// is using to extend kubernetes base api
type CSKubeAPI struct {
	client *csclientset.Clientset
	config *rest.Config
}

var csAPI *CSKubeAPI

func newCSClient(config *rest.Config) (*csclientset.Clientset, error) {
	return csclientset.NewForConfig(config)
}

// CSAPI returns the instance of CSKubeAPI
func CSAPI() *CSKubeAPI {
	return csAPI
}

// NewCSSharedInformerFactory returns a shared informer factory for
// listening to and interacting with containership CRDs
func (k CSKubeAPI) NewCSSharedInformerFactory(t time.Duration) csinformers.SharedInformerFactory {
	return csinformers.NewSharedInformerFactory(k.client, t)
}

// Client returns the clientset which is what is used for getting the different
// CRDs that containership has defined
func (k CSKubeAPI) Client() *csclientset.Clientset {
	return k.client
}

// Config returns the configuration that was used for connecting to
// kubernetes api
func (k CSKubeAPI) Config() *rest.Config {
	return k.config
}
