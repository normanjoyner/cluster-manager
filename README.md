## Containership Agent

### Development Locally

#### Prerequisite
 * Running Kubernetes (minikube, GKE, etc) and have access to the config
 * Golang installed
    * https://golang.org/doc/install
 * glide installed
    * `brew install glide` - or - `curl https://glide.sh/get | sh`


#### Up and Running
```
glide install -v

kubectl apply -f deploy/common

#Runs coordinator
go install cmd/cloud_coordinator/coordinator.go && CONTAINERSHIP_CLOUD_CLUSTER_API_KEY=$CONTAINERSHIP_CLOUD_CLUSTER_API_KEY \
CONTAINERSHIP_CLOUD_ORGANIZATION_ID=$CONTAINERSHIP_CLOUD_ORGANIZATION_ID CONTAINERSHIP_CLOUD_CLUSTER_ID=$CONTAINERSHIP_CLOUD_CLUSTER_ID \
KUBECONFIG=$HOME/.kube/config CONTAINERSHIP_CLOUD_SERVER_PORT=$PORT $GOBIN/coordinator```

#Runs agent
go install cmd/cloud_agent/agent.go && CONTAINERSHIP_CLOUD_CLUSTER_API_KEY=$CONTAINERSHIP_CLOUD_CLUSTER_API_KEY \
CONTAINERSHIP_CLOUD_ORGANIZATION_ID=$CONTAINERSHIP_CLOUD_ORGANIZATION_ID CONTAINERSHIP_CLOUD_CLUSTER_ID=$CONTAINERSHIP_CLOUD_CLUSTER_ID \
KUBECONFIG=$HOME/.kube/config $GOBIN/agent
```


### Running images

#### Set up
* Running Kubernetes (minikube, GKE, etc) and have access to the config
    * `kubectl config use-context minikube` #can be any cluster you wish
* Fill in `containership-env-configmap.yaml` values
* If you are using a GKE cluster, or a cluster with RBAC you will need to give the
service account launching manifests access to talk to the kubernetes api
    * `kubectl apply -f deploy/rbac`

#### Up and Running    
`make coordinator`

`make agent`

### Releases

Rough process is as follows (TODO make this more clear):

1. Create `release-*` branch if it doesn't exist, e.g. `release-2.x`
2. Cherry-pick commits from master to `release-*` branch as necessary (or PR, but ensure no merge commits)
3. Tag, e.g. `git tag -a v2.0.0 -m "Release v2.0.0"` - please don't forget `-a` as it's important we annotate our tags
4. Push release branch and tag to upstream
5. `git checkout v2.0.0`
6. `git clean -fdx` - optional and be careful obviously, but good for sanity
7. `make release` - will fail if you're not on a tag or tree is dirty
9. Push tagged agent and coordinator docker images to DockerHub
10. Bump version to ship in `cloud.api`
