package coordinator

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"k8s.io/api/core/v1"
	kubeinformers "k8s.io/client-go/informers"
	kubeinformerscorev1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	provisioncsv3 "github.com/containership/cluster-manager/pkg/apis/provision.containership.io/v3"
	csclientset "github.com/containership/cluster-manager/pkg/client/clientset/versioned"
	fakecsv3 "github.com/containership/cluster-manager/pkg/client/clientset/versioned/fake"
	csinformers "github.com/containership/cluster-manager/pkg/client/informers/externalversions"
	csinformersprovisionv3 "github.com/containership/cluster-manager/pkg/client/informers/externalversions/provision.containership.io/v3"
	"github.com/containership/cluster-manager/pkg/env"
)

const (
	APIServer         = "kube-apiserver"
	ControllerManager = "kube-controller-manager"
	Scheduler         = "kube-scheduler"
)

var controlPlane = []*v1.Pod{
	{
		ObjectMeta: metav1.ObjectMeta{
			Name:      APIServer,
			Namespace: "kube-system",
			Labels: map[string]string{
				"tier": "control-panel",
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  APIServer,
					Image: APIServer + ":v1.9.2",
				},
			},
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ControllerManager,
			Namespace: "kube-system",
			Labels: map[string]string{
				"tier": "control-panel",
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  ControllerManager,
					Image: ControllerManager + ":v1.9.2",
				},
			},
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name:      Scheduler,
			Namespace: "kube-system",
			Labels: map[string]string{
				"tier": "control-panel",
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  Scheduler,
					Image: Scheduler + ":v1.9.2",
				},
			},
		},
	},
}

func TestGetNextNode(t *testing.T) {
	masterNodeTrue := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "master-true",
			Labels: map[string]string{
				"containership.io/managed":       "true",
				"node-role.kubernetes.io/master": "",
			},
		},
	}

	masterNodeUnmanaged := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "master-true-unmanaged",
			Labels: map[string]string{
				"node-role.kubernetes.io/master": "",
			},
		},
	}

	workerNode := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "worker-master-flag-dne",
			Labels: map[string]string{
				"containership.io/managed": "true",
			},
		},
	}

	masterNodeWithVersion := &v1.Node{
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

	masterNodeWithVersionCopy := masterNodeWithVersion.DeepCopy()
	// Must mutate name in order to place in cache without overwriting
	masterNodeWithVersionCopy.Name = "master-with-version-copy"

	masterNodeWithLabel := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "master-with-label",
			Labels: map[string]string{
				"containership.io/managed":       "true",
				"node-role.kubernetes.io/master": "",
				"custom.label/key":               "value",
			},
		},
	}

	type getNextNodeTest struct {
		name     string
		cluster  []runtime.Object
		input    *provisioncsv3.ClusterUpgrade
		expected runtime.Object
	}

	var nilNode *v1.Node
	tests := []getNextNodeTest{
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
			name: "All nodes at desired version",
			input: &provisioncsv3.ClusterUpgrade{
				Spec: provisioncsv3.ClusterUpgradeSpec{
					Type:          provisioncsv3.UpgradeTypeKubernetes,
					TargetVersion: "v1.9.2",
				},
			},
			cluster: []runtime.Object{
				masterNodeWithVersion,
			},
			expected: nilNode,
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

	for _, test := range tests {
		client, kubeInformerFactory := initializeFakeKubeclient()
		csclientset, csInformerFactory := initializeFakeContainershipClient()
		cupController := NewUpgradeController(
			client, csclientset, kubeInformerFactory, csInformerFactory)

		nodeInformer := kubeInformerFactory.Core().V1().Nodes()
		initializeNodeStore(nodeInformer, test.cluster)
		initializeFakeControlPlane(kubeInformerFactory, test.cluster)

		result := cupController.getNextNode(test.input)
		assert.Equal(t, test.expected, result, test.name)
	}
}

func TestGetNextUpgrade(t *testing.T) {
	upgradeSuccess := &provisioncsv3.ClusterUpgrade{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "upgrade-success",
			Namespace: "containership-core",
			Labels: map[string]string{
				"containership.io/managed": "true",
			},
			CreationTimestamp: metav1.NewTime(time.Unix(0, 0)),
		},
		Spec: provisioncsv3.ClusterUpgradeSpec{
			Type:          provisioncsv3.UpgradeTypeKubernetes,
			TargetVersion: "v1.14.1",
			Status: provisioncsv3.ClusterUpgradeStatusSpec{
				ClusterStatus: provisioncsv3.UpgradeSuccess,
			},
		},
	}

	upgradeFailed := upgradeSuccess.DeepCopy()
	upgradeFailed.Name = "upgrade-failed"
	upgradeFailed.Spec.Status = provisioncsv3.ClusterUpgradeStatusSpec{
		ClusterStatus: provisioncsv3.UpgradeSuccess,
	}

	// Pending upgrades don't have a status set
	upgradePendingTime0 := &provisioncsv3.ClusterUpgrade{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "upgrade-pending-time-0",
			Namespace: "containership-core",
			Labels: map[string]string{
				"containership.io/managed": "true",
			},
			CreationTimestamp: metav1.NewTime(time.Unix(0, 0)),
		},
		Spec: provisioncsv3.ClusterUpgradeSpec{
			Type:          provisioncsv3.UpgradeTypeKubernetes,
			TargetVersion: "v1.14.1",
		},
	}

	upgradePendingTime20 := upgradePendingTime0.DeepCopy()
	upgradePendingTime20.Name = "upgrade-pending-time-20"
	upgradePendingTime20.CreationTimestamp = metav1.NewTime(time.Unix(20, 0))

	upgradePendingTime100 := upgradePendingTime0.DeepCopy()
	upgradePendingTime20.Name = "upgrade-pending-time-100"
	upgradePendingTime100.CreationTimestamp = metav1.NewTime(time.Unix(100, 0))

	type getNextUpgradeTest struct {
		name     string
		upgrades []runtime.Object

		expected    runtime.Object
		shouldError bool
	}

	// TODO test error cases
	// Need this for typed nil
	var nilUpgrade *provisioncsv3.ClusterUpgrade
	tests := []getNextUpgradeTest{
		{
			name:        "No upgrades",
			upgrades:    []runtime.Object{},
			expected:    nilUpgrade,
			shouldError: false,
		},
		{
			name: "No pending upgrades",
			upgrades: []runtime.Object{
				upgradeSuccess,
				upgradeFailed,
			},
			expected:    nilUpgrade,
			shouldError: false,
		},
		{
			name: "Only one pending upgrade",
			upgrades: []runtime.Object{
				upgradePendingTime0,
			},
			expected:    upgradePendingTime0,
			shouldError: false,
		},
		{
			name: "One pending upgrade and multiple finished",
			upgrades: []runtime.Object{
				upgradeSuccess,
				upgradePendingTime0,
				upgradeFailed,
			},
			expected:    upgradePendingTime0,
			shouldError: false,
		},
		{
			name: "Multiple pending upgrades",
			upgrades: []runtime.Object{
				upgradeSuccess,
				upgradePendingTime100,
				upgradeFailed,
				upgradePendingTime0,
				upgradePendingTime20,
			},
			expected:    upgradePendingTime0,
			shouldError: false,
		},
	}

	for _, test := range tests {
		client, kubeInformerFactory := initializeFakeKubeclient()
		csclientset, csInformerFactory := initializeFakeContainershipClient()
		cupController := NewUpgradeController(
			client, csclientset, kubeInformerFactory, csInformerFactory)

		upgradeInformer := csInformerFactory.ContainershipProvision().V3().ClusterUpgrades()
		initializeClusterUpgradeStore(upgradeInformer, test.upgrades)

		result, err := cupController.getNextUpgrade()
		if test.shouldError {
			assert.Error(t, err, result, test.name)
		} else {
			assert.NoError(t, err, result, test.name)
			assert.Equal(t, test.expected, result, test.name)
		}
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

func initializeNodeStore(informer kubeinformerscorev1.NodeInformer, objs []runtime.Object) {
	for _, obj := range objs {
		err := informer.Informer().GetStore().Add(obj)
		fmt.Println(err)
	}
}

func initializeClusterUpgradeStore(informer csinformersprovisionv3.ClusterUpgradeInformer, objs []runtime.Object) {
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
