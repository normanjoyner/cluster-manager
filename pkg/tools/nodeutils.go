package tools

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"

	provisioncsv3 "github.com/containership/cluster-manager/pkg/apis/provision.containership.io/v3"
)

// NodeIsTargetKubernetesVersion checks if the current node version matches the target version
// of the cluster upgrade that is being processed. This only checks that the
// kubelet is up to date on worker nodes, and that on masters kubelet and static pods
// are at the desired version.
// NOTE: this should only be called with upgrades of type Kubernetes
func NodeIsTargetKubernetesVersion(cup *provisioncsv3.ClusterUpgrade, node *corev1.Node, pods []*corev1.Pod) bool {
	kubeletVersion := node.Status.NodeInfo.KubeletVersion
	targetVersion := cup.Spec.TargetVersion

	if _, exists := node.Labels["node-role.kubernetes.io/master"]; !exists {
		return kubeletVersion == targetVersion
	}

	apiVersion := GetNodeAPIVersion(node, pods)
	kubeControllerVersion := GetNodeKubeControllerManagerVersion(node, pods)
	schedulerVersion := GetNodeSchedulerVersion(node, pods)

	return kubeletVersion == targetVersion &&
		apiVersion == targetVersion &&
		kubeControllerVersion == targetVersion &&
		schedulerVersion == targetVersion
}

// GetNodeAPIVersion returns the version of the static pod running the api server
func GetNodeAPIVersion(node *corev1.Node, pods []*corev1.Pod) string {
	return getPodforNodeByContainerName("kube-apiserver", node.Name, pods)
}

// GetNodeKubeControllerManagerVersion returns the version of the static pod running
// the kube controller manager
func GetNodeKubeControllerManagerVersion(node *corev1.Node, pods []*corev1.Pod) string {
	return getPodforNodeByContainerName("kube-controller-manager", node.Name, pods)
}

// GetNodeSchedulerVersion returns the version of the static pod running the scheduler
func GetNodeSchedulerVersion(node *corev1.Node, pods []*corev1.Pod) string {
	return getPodforNodeByContainerName("kube-scheduler", node.Name, pods)
}

// getPodforNodeByContainerName uses the name passed in concatenated with the
// node name to get a pod by the name. It then returns the image that is being
// used by the container with the same name that is passed in.
func getPodforNodeByContainerName(name string, nodeName string, pods []*corev1.Pod) string {
	for _, pod := range pods {
		if pod.Name != fmt.Sprintf("%s-%s", name, nodeName) {
			continue
		}

		for _, container := range pod.Spec.Containers {
			if container.Name != name {
				continue
			}

			return getImageVersion(container.Image)
		}
	}

	return ""
}

func getImageVersion(image string) string {
	parts := strings.Split(image, ":")
	// get the last part of the array
	// ex.  <YOUR-DOMAIN>:8080/test-image:tag
	version := parts[len(parts)-1]

	if version != "" {
		return version
	}

	return "latest"
}

// NodeIsReady returns true if the given node has a Ready status, else false.
// See https://kubernetes.io/docs/concepts/nodes/node/#condition for more info.
func NodeIsReady(node *corev1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}
