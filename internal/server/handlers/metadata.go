package handlers

import (
	"net/http"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	typev1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"

	"github.com/containership/cloud-agent/internal/envvars"
)

type node struct {
	typev1.NodeSystemInfo
	NodeID string `json: "nodeID"`
}

type containershipClusterMetadata struct {
	ClusterID      string `json:"cluster_id"`
	OrganizationID string `json:"organization_id"`
}
type metadata struct {
	Containership containershipClusterMetadata `json:"containership"`
	Timestamp     time.Time                    `json:"timestamp"`
	Nodes         []node
}

func getNodes() ([]node, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	nodes, err := clientset.CoreV1Client.Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	allNodes := make([]node, 0)
	for _, n := range nodes.Items {
		allNodes = append(allNodes, node{n.Status.NodeInfo, "nodeid"})
	}

	return allNodes, nil
}

// Get returns Containership and node metadata
func (meta *Metadata) Get(w http.ResponseWriter, r *http.Request) {
	nodes, err := getNodes()

	if err != nil {
		respondWithError(w, 500, err.Error())
		return
	}

	m := &metadata{
		Containership: containershipClusterMetadata{
			ClusterID:      envvars.GetClusterID(),
			OrganizationID: envvars.GetOrganizationID(),
		},
		Timestamp: time.Now(),
		Nodes:     nodes,
	}

	respondWithJSON(w, http.StatusOK, m)
}
