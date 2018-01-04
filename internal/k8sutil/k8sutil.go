package k8sutil

import (
	"log"
	"time"

	"github.com/containership/cloud-agent/internal/envvars"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	kubeinformers "k8s.io/client-go/informers"
)

// KubeAPI defines an object to be able to easily
// talk with kubernetes, and store needed information about how
// we are talking to kubernetes
type KubeAPI struct {
	client *kubernetes.Clientset
	config *rest.Config
}

var kubeAPI *KubeAPI

func init() {
	var err error
	config, err := determineConfig()
	if err != nil {
		log.Println(err.Error())
		return
	}

	clientset, err := newKubeClient(config)
	if err != nil {
		log.Println(err.Error())
	}

	csclientset, err := newCSClient(config)
	if err != nil {
		log.Println(err.Error())
	}

	kubeAPI = &KubeAPI{clientset, config}
	csAPI = &CSKubeAPI{csclientset, config}
}

// determineConfig determines if we are running in a cluster or out side
// and gets the appropriate configuration to talk with kubernetes
func determineConfig() (*rest.Config, error) {
	kubeconfigPath := envvars.GetKubeconfig()
	var config *rest.Config
	var err error

	// determine whether to use in cluster config or out of cluster config
	// if kuebconfigPath is not specified, default to in cluster config
	// otherwise, use out of cluster config
	if kubeconfigPath == "" {
		log.Println("Using in cluster k8s config")
		config, err = rest.InClusterConfig()
	} else {
		log.Printf("Using out of cluster k8s config: %s", kubeconfigPath)

		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	}

	if err != nil {
		return nil, err
	}

	return config, nil
}

func newKubeClient(config *rest.Config) (*kubernetes.Clientset, error) {
	return kubernetes.NewForConfig(config)
}

// API returns an instance of the KubeAPI
func API() *KubeAPI {
	return kubeAPI
}

// NewKubeSharedInformerFactory returns the shared informer factory
// for watching kubernetes resource events
func (k KubeAPI) NewKubeSharedInformerFactory(t time.Duration) kubeinformers.SharedInformerFactory {
	return kubeinformers.NewSharedInformerFactory(k.Client(), t)
}

// Client returns the client set that is used to interact with
// the objects that kubernetes has defined
func (k KubeAPI) Client() *kubernetes.Clientset {
	return k.client
}

// Config returns the configuration that was used for connecting to
// kubernetes api
func (k KubeAPI) Config() *rest.Config {
	return k.config
}

// GetNodes returns all nodes running the kublet in the kubernetes cluster
func (k KubeAPI) GetNodes() (*corev1.NodeList, error) {
	nodes, err := k.Client().CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		log.Println("Error getting nodes: ", err)
		return nil, err
	}

	return nodes, nil
}

// GetNamespaces returns all namespaces from the kubernetes cluster
func (k KubeAPI) GetNamespaces() (*corev1.NamespaceList, error) {
	namespaces, err := k.Client().CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		log.Println("Error getting namespaces: ", err)
		return nil, err
	}

	return namespaces, nil
}
