package k8sutil

import (
	"k8s.io/client-go/rest"

	"github.com/containership/cluster-manager/pkg/constants"
	v1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	extclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KubeExtensionsAPI provides access to the apiextensions API group
type KubeExtensionsAPI struct {
	client *extclientset.Clientset
	config *rest.Config
}

var kubeExtensionsAPI *KubeExtensionsAPI

// newKubeExtensionsAPI constructs a new apiextensions clientset
// for the given config
func newKubeExtensionsAPI(config *rest.Config) (*extclientset.Clientset, error) {
	return extclientset.NewForConfig(config)
}

// ExtensionsAPI returns the apiextensions API
func ExtensionsAPI() *KubeExtensionsAPI {
	return kubeExtensionsAPI
}

// Client returns the apiextensions client
func (k KubeExtensionsAPI) Client() *extclientset.Clientset {
	return k.client
}

// GetContainershipCRDs gets all Containership CRDs
func (k KubeExtensionsAPI) GetContainershipCRDs() (*v1beta1.CustomResourceDefinitionList, error) {
	return k.Client().ApiextensionsV1beta1().
		CustomResourceDefinitions().List(metav1.ListOptions{
		LabelSelector: constants.BaseContainershipManagedLabelString,
	})
}

// DeleteCRD deletes the CRD with the given name
func (k KubeExtensionsAPI) DeleteCRD(name string) error {
	return k.Client().ApiextensionsV1beta1().
		CustomResourceDefinitions().Delete(name, &metav1.DeleteOptions{})
}
