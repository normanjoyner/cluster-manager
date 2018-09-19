package coordinator

import (
	"fmt"
	"testing"

	"github.com/containership/cloud-agent/pkg/env"
	"github.com/stretchr/testify/assert"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"k8s.io/api/core/v1"
	kubeinformers "k8s.io/client-go/informers"
	kubeinformerscorev1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	provisioncsv3 "github.com/containership/cloud-agent/pkg/apis/provision.containership.io/v3"
	csclientset "github.com/containership/cloud-agent/pkg/client/clientset/versioned"
	fakecsv3 "github.com/containership/cloud-agent/pkg/client/clientset/versioned/fake"
	csinformers "github.com/containership/cloud-agent/pkg/client/informers/externalversions"
)

type buildLabelTest struct {
	name     string
	cluster  []runtime.Object
	input    *provisioncsv3.ClusterUpgrade
	expected runtime.Object
}

var masterNodeTrue = &v1.Node{
	ObjectMeta: metav1.ObjectMeta{
		Name: "master-true",
		Labels: map[string]string{
			"containership.io/managed":       "true",
			"node-role.kubernetes.io/master": "",
		},
	},
}

var masterNodeUnmanaged = &v1.Node{
	ObjectMeta: metav1.ObjectMeta{
		Name: "master-true-unmanaged",
		Labels: map[string]string{
			"node-role.kubernetes.io/master": "",
		},
	},
}

var workerNode = &v1.Node{
	ObjectMeta: metav1.ObjectMeta{
		Name: "worker-master-flag-dne",
		Labels: map[string]string{
			"containership.io/managed": "true",
		},
	},
}

var masterNodeWithVersion = &v1.Node{
	ObjectMeta: metav1.ObjectMeta{
		Name: "master-with-version",
		Labels: map[string]string{
			"containership.io/managed":       "true",
			"node-role.kubernetes.io/master": "",
		},
	},
	Status: v1.NodeStatus{
		NodeInfo: v1.NodeSystemInfo{
			KubeletVersion: "v1.9.2",
		},
	},
}

var masterNodeWithLabel = &v1.Node{
	ObjectMeta: metav1.ObjectMeta{
		Name: "master-with-label",
		Labels: map[string]string{
			"containership.io/managed":       "true",
			"node-role.kubernetes.io/master": "",
			"custom.label/key":               "value",
		},
	},
}

const APIServer = "kube-apiserver"
const ControllerManager = "kube-controller-manager"
const Scheduler = "kube-scheduler"

var controlPlane = []*v1.Pod{
	&v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: APIServer,
			Namespace: "kube-system",
			Labels: map[string]string{
				"tier":       "control-panel",
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name: APIServer,
					Image: APIServer + ":v1.9.2",
				},
			},
		},
	},
	&v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: ControllerManager,
			Namespace: "kube-system",
			Labels: map[string]string{
				"tier":       "control-panel",
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name: ControllerManager,
					Image: ControllerManager + ":v1.9.2",
				},
			},
		},
	},
	&v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: Scheduler,
			Namespace: "kube-system",
			Labels: map[string]string{
				"tier":       "control-panel",
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name: Scheduler,
					Image: Scheduler + ":v1.9.2",
				},
			},
		},
	},
}

var tests = []buildLabelTest{
	{
		name: "Master node next",
		input: &provisioncsv3.ClusterUpgrade{
			Spec: provisioncsv3.ClusterUpgradeSpec{
				Type:          provisioncsv3.UpgradeTypeKubernetes,
				TargetVersion: "v1.9.2",
			},
		},
		cluster: []runtime.Object{
			masterNodeTrue,
			workerNode,
		},
		expected: masterNodeTrue,
	},
	{
		name: "Master node unmanaged",
		input: &provisioncsv3.ClusterUpgrade{
			Spec: provisioncsv3.ClusterUpgradeSpec{
				Type:          provisioncsv3.UpgradeTypeKubernetes,
				TargetVersion: "v1.9.2",
			},
		},
		cluster: []runtime.Object{
			masterNodeUnmanaged,
			workerNode,
		},
		expected: workerNode,
	},
	{
		name: "Master node at desired version. return worker",
		input: &provisioncsv3.ClusterUpgrade{
			Spec: provisioncsv3.ClusterUpgradeSpec{
				Type:          provisioncsv3.UpgradeTypeKubernetes,
				TargetVersion: "v1.9.2",
			},
		},
		cluster: []runtime.Object{
			masterNodeWithVersion,
			workerNode,
		},
		expected: workerNode,
	},
	{
		name: "Master node at desired version. return next master",
		input: &provisioncsv3.ClusterUpgrade{
			Spec: provisioncsv3.ClusterUpgradeSpec{
				Type:          provisioncsv3.UpgradeTypeKubernetes,
				TargetVersion: "v1.9.2",
			},
		},
		cluster: []runtime.Object{
			masterNodeWithVersion,
			masterNodeTrue,
		},
		expected: masterNodeTrue,
	},
	{
		name: "Get node with label selector",
		input: &provisioncsv3.ClusterUpgrade{
			Spec: provisioncsv3.ClusterUpgradeSpec{
				TargetVersion: "v1.9.2",
				LabelSelector: []provisioncsv3.LabelSelectorSpec{
					{
						Label:    "custom.label/key",
						Operator: "=",
						Value:    []string{"value"},
					},
				},
			},
		},
		cluster: []runtime.Object{
			masterNodeWithLabel,
			masterNodeTrue,
		},
		expected: masterNodeWithLabel,
	},
}

func TestGetNextNode(t *testing.T) {
	for _, test := range tests {
		client, kubeInformerFactory := initializeFakeKubeclient()
		csclientset, csInformerFactory := initializeFakeContainershipClient()
		cupController := NewUpgradeController(
			client, csclientset, kubeInformerFactory, csInformerFactory)

		nodeInformer := kubeInformerFactory.Core().V1().Nodes()
		initializeStore(nodeInformer, test.cluster)
		initializeFakeControlPlane(kubeInformerFactory, test.cluster)

		result := cupController.getNextNode(test.input)
		assert.Equal(t, test.expected, result, test.name)
	}
}

func initializeFakeKubeclient() (kubernetes.Interface, kubeinformers.SharedInformerFactory) {
	client := &fake.Clientset{}
	interval := env.CoordinatorInformerSyncInterval()

	kubeInformerFactory = kubeinformers.NewSharedInformerFactory(client, interval)

	return client, kubeInformerFactory
}

func initializeFakeContainershipClient() (csclientset.Interface, csinformers.SharedInformerFactory) {
	csclientset := fakecsv3.NewSimpleClientset()
	interval := env.CoordinatorInformerSyncInterval()

	csInformerFactory = csinformers.NewSharedInformerFactory(csclientset, interval)

	return csclientset, csInformerFactory
}

func initializeStore(informer kubeinformerscorev1.NodeInformer, objs []runtime.Object) {
	for _, obj := range objs {
		err := informer.Informer().GetStore().Add(obj)
		fmt.Println(err)
	}
}

func initializeFakeControlPlane(kubeInformerFactory kubeinformers.SharedInformerFactory, cluster []runtime.Object) {
	podInformer := kubeInformerFactory.Core().V1().Pods()
	for _, obj := range cluster {
		node, ok := obj.(*v1.Node)
		if !ok {
			continue
		}

		if _, ok := node.Labels["node-role.kubernetes.io/master"]; !ok {
			continue
		}

		nodeName := node.Name
		objs := make([]runtime.Object, 0)
		for _, pod := range controlPlane {
			podC := pod.DeepCopy()
			podC.Name = podC.Name + "-" + nodeName
			objs = append(objs, podC)
		}

		initializePodStore(podInformer, objs)
	}
}

func initializePodStore(informer kubeinformerscorev1.PodInformer, objs []runtime.Object) {
	for _, obj := range objs {
		err := informer.Informer().GetStore().Add(obj)
		fmt.Println(err)
	}
}
