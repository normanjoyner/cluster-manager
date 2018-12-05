package k8sutil

import (
	"time"

	"k8s.io/client-go/rest"

	cerebralclientset "github.com/containership/cerebral/pkg/client/clientset/versioned"
	cerebralinformers "github.com/containership/cerebral/pkg/client/informers/externalversions"
)

// CerebralKubeAPI is an object for interacting with the CRDs cerebral
// is using to extend kubernetes base api
type CerebralKubeAPI struct {
	client *cerebralclientset.Clientset
	config *rest.Config
}

var cerebralAPI *CerebralKubeAPI

func newCerebralClient(config *rest.Config) (*cerebralclientset.Clientset, error) {
	return cerebralclientset.NewForConfig(config)
}

// CerebralAPI returns the instance of CerebralKubeAPI
func CerebralAPI() *CerebralKubeAPI {
	return cerebralAPI
}

// NewCerebralSharedInformerFactory returns a shared informer factory for
// listening to and interacting with containership CRDs
func (k CerebralKubeAPI) NewCerebralSharedInformerFactory(t time.Duration) cerebralinformers.SharedInformerFactory {
	return cerebralinformers.NewSharedInformerFactory(k.client, t)
}

// Client returns the clientset which is what is used for getting the different
// CRDs that containership has defined
func (k CerebralKubeAPI) Client() *cerebralclientset.Clientset {
	return k.client
}

// Config returns the configuration that was used for connecting to
// kubernetes api
func (k CerebralKubeAPI) Config() *rest.Config {
	return k.config
}
