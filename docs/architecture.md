# Cluster Manager Architecture

## Overview

The cluster manager consists of two main components: a coordinator which runs as a Kubernetes [Deployment][deployment] (with a single replica) and an agent which runs as a Kubernetes [Daemonset][daemonset] (on all nodes including masters).
A high-level overview of the responsibilities of these two overarching components is available in the [main README][main-readme].
In this document, we'll do a deep-dive into the individual components of both the coordinator and the agent.

## Coordinator

The coordinator is responsible for synchronizing and reconciling Containership cloud resources (e.g. registries, plugins) as well as orchestrating Kubernetes cluster upgrades.
It differs from the agent in that it only operates on cluster-level resources.

At a high level, the coordinator is composed of the cloud synchronizer and numerous Kubernetes controllers that have a relatively small, well-bounded scope for ensuring the desired state of specific resources.

### Cloud Synchronizer

The cloud synchronizer is used to synchronize Containership cloud resources to Kubernetes Custom Resources (CRs) that are defined by [Custom Resource Definitions (CRDs)][crd].
It acts as a clean division between the Containership way of representing and operating on resources and the Kubernetes way.
For this reason, cloud synchronizer is the single component of the coordinator that is not technically a Kubernetes controller itself.
However, it heavily leverages the concept of a Kubernetes controller:

- It constantly reconciles the CRs on the cluster.
- If a CR does not exist, then it creates it.
- If a CR exists that is not expected (e.g. because the cloud resource was deleted), it deletes it.

#### Resource Sync Controllers

Every resource type that can be synced to a CR has its own "sync controller".
Each individual resource sync controller requests its resources from cloud and translate them to CRs.
The sync controllers are all very similar in construction and operation.
They are all registered with the cloud synchronizer which behaves as the top-level orchestrator.

### Upgrade Controller

The coordinator upgrade controller orchestrates Kubernetes upgrades at the cluster level.
Because the work it performs is more involved, it's detailed in a separate [Kubernetes upgrade][upgrade-doc] document.

### Resource Controllers

The resource controllers watch for events on the CRs created by the resource _sync_ controllers and reconcile resources within the cluster accordingly.

#### Containership Controller

The Containership controller assists the registry controller (below) by ensuring the `containership` service account exists in all namespaces.

#### Registry Controller

The registry controller is responsible for creating [image pull secrets][image-pull-secrets] in all namespaces for all container registries added through Containership cloud.
It also attaches these secrets to the required service accounts in all namespaces.

#### Label Controllers

A cluster label is a label that is applied to every node within a Containership cluster.
A node pool label is a label that is applied to every node within a Containership node pool.
The label controllers watch for events on `ClusterLabel` and `NodePoolLabel` CRs and ensure that each node in the cluster has the expected cluster and/or node pool labels.

#### Node Pool Label Controller

The node pool label controller watches for events on `NodePoolLabel` CRs and ensures that each node pool label on each node matches the desired set from cloud.

#### Plugin Controller

The plugin controller is responsible for synchronizing plugins (e.g. Prometheus for metrics, Cerebral for autoscaling, etc.) to the cluster.

### RBAC Controllers

The authorization role and authorization role binding controllers synchronize RBAC from cloud to native Kubernetes RBAC resources.
This is how Kubernetes permissions are granted to Containership users and teams.

## Agent

The agent is responsible for synchronizing node-level resources (i.e. users and SSH keys) and performing Kubernetes upgrades for a single node.

### Upgrade Controller

The agent upgrade controller performs Kubernetes upgrades on an individual node.
It is detailed in a separate [Kubernetes upgrade][upgrade-doc] document.

### User Controller

The user controller watches for events on Containership `User` CRs and reconciles the SSH keys on each node accordingly.
It also acts on events from an `authorized_keys` file watcher.

#### Authorized Keys File Watcher

SSH keys are synchronized to an `authorized_keys` file on the node's filesystem. 
In order to detect unexpected changes to the `authorized_keys` file and overwrite them, a file watcher based on [`fsnotify`][fsnotify] is run as a separate routine within the user controller.

[crd]: https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions
[daemonset]: https://kubernetes.io/docs/concepts/workloads/controllers/daemonset/
[deployment]: https://kubernetes.io/docs/concepts/workloads/controllers/deployment/
[image-pull-secrets]: https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/
[main-readme]: ../README.md
[upgrade-doc]: kubernetes_upgrade.md
[fsnotify]: https://github.com/fsnotify/fsnotify
