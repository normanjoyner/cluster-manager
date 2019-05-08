# Upgrading and Downgrading Kubernetes Versions

## Overview

`cluster-manager` is able to trigger and orchestrate upgrades and downgrades of the Kubernetes version on master and worker node pools through the use of a [CustomResourceDefinition (CRD)][crd].

Only Containership Kubernetes Engine (CKE) clusters may be upgraded using this mechanism.
Typically, upgrades are simply triggered through the [Containership UI][containership_ui] or via [`csctl`][csctl].
This posts a `ClusterUpgrade` to the cluster, causing the `cluster-manager` to kick off an upgrade.

Upgrades are performed in-place on a node-by-node basis.
This is in contrast to e.g. `kops` which performs Kubernetes upgrades by replacing nodes.
In-place upgrades are less disruptive than node replacements, and also typically much faster.
By performing upgrades in this way, users do not have to worry about complications such as Persistent Volume Claims (PVCs) being released and delays associated with reattaching them.
See the [upgrade process](#upgrade-process) section below for more details on how the in-place upgrades work.

Downgrades are supported using exactly the same mechanism.
Only rollbacks of failed upgrades are supported through the UI.

## ClusterUpgrade CRD

The `ClusterUpgrade` CRD manifest lives [here][clusterupgrade_crd].
Below is an example `ClusterUpgrade` CR:

```
apiVersion: provision.containership.io/v3
  kind: ClusterUpgrade
  metadata:
    labels:
      containership.io/managed: "true"
      containership.io/node-pool-id: f643e136-1c75-4b01-a5dd-04bce9103162
    name: f595244e-c2a6-49f9-9b54-7050c258aa8f
    namespace: containership-core
  spec:
    addedAt: "1547475212"
    description: Upgrading to 1.12.3
    id: f595244e-c2a6-49f9-9b54-7050c258aa8f
    labelSelector:
    - label: containership.io/node-pool-id
      operator: ==
      value:
      - f643e136-1c75-4b01-a5dd-04bce9103162
    nodeTimeoutSeconds: 600
    targetVersion: v1.12.3
    type: kubernetes
  status:
    clusterStatus: Success
    currentNode: ""
    currentStartTime: ""
    nodeStatuses:
      6dc90d6e-8250-4dfe-9369-6c7aef99f9d9: Success
```

#### Spec Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| addedAt | string | A string representation of the Unix time at which this ClusterUpgrade was added |
| description | string | A description for this upgrade |
| id | string | An ID for this upgrade which must be unique across all upgrades for a Cluster|
| labelSelector | object | A label selector used to select all nodes that should be upgraded
| labelSelector.label | string | The label key for selection
| labelSelector.operator | string | The operator for selection. Valid operators are listed [here][label_selector_operators]
| labelSelector.value | string | The label value for selection
| nodeTimeoutSeconds | int | The time in seconds after which a single node upgrade should be considered failed and should be marked as such and halted
| targetVersion | string | The version to upgrade to. Must include the leading `v`.
| type | string | The type of upgrade. The only supported value is `kubernetes`.

#### Status Fields

| Field | Type | Description |
| ----- | ---- | ----------- |
| clusterStatus | string | The overall cluster upgrade status. Valid values are: `InProgress`, `Success`, `Failed`
| currentNode | string | The `id` of the current node being upgraded, if any
| currentNodeStartTime | string | If `currentNode` is non-empty, this is a string representation of the Unix time at which the `currentNode` began upgrading
| nodeStatuses | object  | A map of individual node `id` to node status. Valid status values are the same as for `clusterStatus`.

*Note*: The `status` field is not a [`status` subresource][status_subresource] but rather a plain CR field.

## Upgrade Process

As a refresher, the `cluster-manager` is composed of the [coordinator][coordinator] Deployment and [agent][agent] DaemonSet.
The `coordinator` orchestrates the overall cluster upgrade, while the `agent` performs the actual node upgrade for each node it is running on.

The `coordinator` watches for new `ClusterUpgrade` CRs.
When a new CR is created, it sanity checks the values.
If the CR is well-formed (i.e. accepted by Kubernetes, which is how the `coordinator` sees it in the first place) but not valid, it will immediately be marked as `Failed`.
If the CR is valid, the `clusterStatus` will be updated to `InProgress` and the first node will be chosen to upgrade.

When an `agent` sees that its node has been marked as `InProgress` by the `coordinator`, it kicks off a Kubernetes upgrade for the node it's running on.
The upgrade is performed by downloading and running a script requested from Containership Cloud.

Retries are performed within the upgrade script for critical steps.
Should a failure occur, care is taken to ensure that the system is reset to its original state.
In other words, the upgrade script is idempotent.

### Master Node Upgrades

Since CKE clusters are `kubeadm`-based, the upgrade script is a fairly straightforward invocation of `kubeadm upgrade` along with any necessary configuration changes.
After the control plane components are upgraded, an updated `kubelet` is put into place and the `kubelet` is restarted.

### Worker Node Upgrades

Worker nodes do not run control plane components, so only the `kubelet` must be upgraded.
An updated `kubelet` is put into place along with any configuration updates and the `kubelet` is restarted.

### Handling Failures

#### Coordinator Restarts

The `coordinator` may go down and be rescheduled throughout the upgrade process.
This is expected and does not affect the upgrade process.
Since all upgrade state is stored in the `ClusterUpgrade` CR, the `coordinator` simply picks up where it left off when it starts back up.

#### Node Upgrade Timeouts

If an individual node upgrade does not complete in time, the `coordinator` will mark that node as `Failed` in the `nodeStatuses` object.
This node is said to have timed out.
In the event of a node timeout, the entire upgrade process continues as usual onto the next node.
The process is not halted.
At the end of the upgrade process, the `coordinator` will mark `clusterStatus` as `Failed` if any individual node timed out.

#### Retrying a Failed Upgrade

In the unlikely event that a cluster upgrade fails, the `clusterStatus` will be marked as `Failed`.
In this case, the upgrade may be retried by simply posting a new `ClusterUpgrade` with the same `labelSelector` and `targetVersion`.
The `id` *must* be unique across all `ClusterUpgrade`s for a cluster, even if it is considered a retry.

This can be performed through the [Containership UI][containership_ui] simply by pressing the retry button in the node pool upgrade view.

## Assumptions and Limitations

There are several assumptions made during the upgrade process.
If these assumptions are not accurate, then an upgrade may fail or there may be undefined behavior.
If upgrading through the [Containership UI][containership_ui], it is not possible to kick off an upgrade for which the assumptions are not met.

There are several version skew requirements relating to upgrades.
These are well-documented by the `kubeadm` and core `Kubernetes` project.

#### Version Skew Requirements

- It is not possible to skip a minor version when upgrading
  - If upgrading from `1.X.x` to `1.Y.y`, `|Y - X| <= 1` is the only path supported
- The `kubelet` versions within the cluster must not exceed the control plane version
- The `kubelet` versions may not lag behind the control plane version by more than two minor versions

For CKE clusters, the requirements relating to the version skew between the `kubelet` and control plane equate equate to the version skew between master nodes and worker nodes.
This is because a master node upgrade is marked as successful if and only if the control plane and `kubelet` upgrades both succeed.

[containership_ui]: https://cloud.containership.io
[clusterupgrade_crd]: /deploy/crd/containership-clusterupgrade-crd.yaml
[csctl]: https://github.com/containership/csctl
[coordinator]: https://github.com/containership/cluster-manager#coordinator
[agent]: https://github.com/containership/cluster-manager#agent
[crd]: https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions
[status_subresource]: https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/#status-subresource
[label_selector_operators]: https://github.com/kubernetes/apimachinery/blob/a1e35b7/pkg/selection/operator.go#L23-L33
