# Containership Cluster Manager

[![Build Status](https://travis-ci.org/containership/cluster-manager.svg?branch=master)](https://travis-ci.org/containership/cluster-manager)
[![Go Report Card](https://goreportcard.com/badge/github.com/containership/cluster-manager)](https://goreportcard.com/report/github.com/containership/cluster-manager)
[![codecov](https://codecov.io/gh/containership/cluster-manager/branch/master/graph/badge.svg)](https://codecov.io/gh/containership/cluster-manager)

## Overview

The [Containership][containership] cluster manager is a plugin that enables deep integrations with Containership Cloud for clusters provisioned using Containership Kubernetes Engine (CKE) as well as existing clusters that have been attached to the platform.

### Features

#### Core Features

There are many core features that apply to both CKE clusters and attached clusters.

- [Plugin management][plugins]
- [SSH key synchronization][ssh-keys]
- [Container registry credential synchronization][registries]

#### CKE Features

These features are available only for clusters provisioned using CKE.

- Kubernetes (and etcd) upgrade management

## Structure

The Containership cluster manager is composed of two primary components: the `cloud-coordinator` and `cloud-agent`.

### Coordinator

The Containership coordinator runs as a Kubernetes [Deployment][deployment] with a single replica.
It performs synchronization and reconciliation of Containership cloud resources (e.g. registries, plugins).
It is also responsible for orchestrating Kubernetes cluster upgrades.

### Agent

The Containership agent runs as a Kubernetes [DaemonSet][daemonset] on all nodes, including masters.
It performs synchronization and reconciliation of host-level resources such as SSH keys.
It is also responsible for performing Kubernetes upgrades of individual nodes.

## Releases and Compatibility

Containership releases follow the [semver](https://semver.org/) format.
Similar to other projects in the Kubernetes ecosystem, a new major version of the cluster manager is released for every Kubernetes minor version.
Also in line with the Kubernetes ecosystem, Containership supports a sliding window of 3 minor versions of Kubernetes.
Please refer to the following compatibility matrix for specific compatibility information.
Please also refer to the [official changelog][cluster-management-changelog] for release notes.

### Compatibility Matrix

Support for previous Kubernetes versions can be inferred by using the sliding window.
For example, `cluster-manager` 2.x supports Kubernetes versions 1.8.x - 1.10.x.

|                     | Kubernetes 1.13.x | Kubernetes 1.12.x | Kubernetes 1.11.x | Kubernetes 1.10.x |
|---------------------|-------------------|-------------------|-------------------|-------------------|
| cluster-manager 5.x | ✓                 | ✓                 | ✓                 | ✗                 |
| cluster-manager 4.x | ✗                 | ✓                 | ✓                 | ✓                 |
| cluster-manager 3.x | ✗                 | ✗                 | ✓                 | ✓                 |
| cluster-manager 2.x | ✗                 | ✗                 | ✗                 | ✓                 |


## Contributing
Thank you for your interest in this project and for your interest in contributing! Feel free to open issues for feature requests, bugs, or even just questions - we love feedback and want to hear from you.

PRs are also always welcome! However, if the feature you're considering adding is fairly large in scope, please consider opening an issue for discussion first.

[containership]: https://containership.io/
[plugins]: https://docs.containership.io/getting-started/attach-cluster/containership-plugins
[ssh-keys]: https://docs.containership.io/getting-started/account-and-organization/managing-ssh-keys
[registries]: https://docs.containership.io/getting-started/account-and-organization/managing-image-registry-credentials
[deployment]: https://kubernetes.io/docs/concepts/workloads/controllers/deployment/
[daemonset]: https://kubernetes.io/docs/concepts/workloads/controllers/daemonset/
[cluster-management-changelog]: https://github.com/containership/plugins-changelog/blob/master/cluster_management/containership/CHANGELOG.md
