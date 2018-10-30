# make file mount of internal code
`make mount`

# deploy DaemonSet, and Deployment which mounts in internal code from mount
`kubectl apply -f deploy/development`

# get the pods ids
`COORDINATOR_POD=$(kubectl get pods -o go-template --template '{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}' | grep "containership-coordinator-*")`
`AGENT_POD=$(kubectl get pods -o go-template --template '{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}' | grep "containership-agent-*")`

# Exec into coordinator pod
`kubectl exec -it $COORDINATOR_POD -- /bin/sh`
`cd /go/src/github.com/containership/cluster-manager/`

# Exec into agent pod
`kubectl exec -it $AGENT_POD -- /bin/sh`
`cd /go/src/github.com/containership/cluster-manager/`

From here you can develop locally,
and inside the kubernetes pod run `go run cmd/(cloud_agent|cloud_coordinator)/main.go`
depending which service is being worked on
