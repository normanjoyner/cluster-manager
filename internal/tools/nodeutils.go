package tools

import (
	corev1 "k8s.io/api/core/v1"

	provisioncsv3 "github.com/containership/cloud-agent/pkg/apis/provision.containership.io/v3"
)

// NodeIsTargetKubernetesVersion checks if the current node version matches the target version
// of the cluster upgrade that is being processed. This only checks that the
// kubelet is up to date, and does not check the static pods.
// TODO: later we should consider doing this check in a safer/more reliable way
func NodeIsTargetKubernetesVersion(cup *provisioncsv3.ClusterUpgrade, node *corev1.Node) bool {
	return node.Status.NodeInfo.KubeletVersion == cup.Spec.TargetKubernetesVersion
}
