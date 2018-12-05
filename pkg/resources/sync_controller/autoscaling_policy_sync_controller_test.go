package synccontroller

import (
	"testing"
	"time"

	"k8s.io/client-go/kubernetes/fake"

	fakecerebral "github.com/containership/cerebral/pkg/client/clientset/versioned/fake"
	cerebralinformers "github.com/containership/cerebral/pkg/client/informers/externalversions"
	"github.com/stretchr/testify/assert"
)

func initalizeFakeAutoscalingPolicyController() *AutoscalingPolicySyncController {
	client := &fake.Clientset{}
	cerebralclient := fakecerebral.NewSimpleClientset()

	cerebralInformerFactory := cerebralinformers.NewSharedInformerFactory(cerebralclient, 30*time.Second)
	return NewAutoscalingPolicyController(client, cerebralclient, cerebralInformerFactory)
}

// NewAutoscalingPolicyController(kubeclientset kubernetes.Interface, clientset cerebral.Interface, cerebralInformerFactory cerebralinformers.SharedInformerFactory) *AutoscalingPolicySyncController
func TestNewAutoscalingPolicyController(t *testing.T) {
	apc := initalizeFakeAutoscalingPolicyController()

	// test to make sure new creating a autoscaling controller is being
	// created and returned
	assert.Equal(t, autoscalingPolicySyncControllerName, apc.syncController.name)
}
