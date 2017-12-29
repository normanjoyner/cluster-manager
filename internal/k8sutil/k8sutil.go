package k8sutil

import (
	"flag"
	"log"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var clientset *kubernetes.Clientset

func init() {
	var kubeconfig *string
	kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	flag.Parse()

	var err error
	clientset, err = newClient(*kubeconfig)
	if err != nil {
		log.Fatal(err.Error())
	}
}

func newClient(kubeconfigPath string) (*kubernetes.Clientset, error) {
	var client *kubernetes.Clientset

	// determine whether to use in cluster config or out of cluster config
	// if kuebconfigPath is not specified, default to in cluster config
	// otherwise, use out of cluster config
	if kubeconfigPath == "" {
		log.Println("Using in cluster k8s config")
		config, err := rest.InClusterConfig()

		if err != nil {
			return nil, err
		}

		client, err = kubernetes.NewForConfig(config)

		if err != nil {
			return nil, err
		}
	} else {
		log.Println("Using out of cluster k8s config: %s", kubeconfigPath)
		config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)

		if err != nil {
			return nil, err
		}

		client, err = kubernetes.NewForConfig(config)

		if err != nil {
			return nil, err
		}
	}

	return client, nil
}

// GetNodes returns all nodes running the kublet in the kubernetes cluster
func GetNodes() (*corev1.NodeList, error) {
	nodes, err := clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		log.Println("Error getting nodes: ", err)
		return nil, err
	}

	return nodes, nil
}

// GetNamespaces returns all namespaces from the kubernetes cluster
func GetNamespaces() (*corev1.NamespaceList, error) {
	namespaces, err := clientset.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		log.Println("Error getting namespaces: ", err)
		return nil, err
	}

	return namespaces, nil
}
