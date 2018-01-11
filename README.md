## Containership Agent

### Development Locally

#### Prerequisite
 * Running kubernetes (minikube, gke, etc) and have access to the config
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
* Running kubernetes (minikube, gke, etc) and have access to the config
    * `kubectl config use-context minikube` #can be any cluster you wish
* Fill in `containership-env-configmap.yaml` values
* If you are using a GKE cluster, or a cluster with RBAC you will need to give the
service account launching manifests access to talk to the kubernetes api
    * `kubectl apply -f deploy/rbac`

#### Up and Running    
`make coordinator`

`make agent`
