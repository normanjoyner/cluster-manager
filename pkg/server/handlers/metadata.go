package handlers

import (
	"net/http"
	"time"

	"github.com/containership/cluster-manager/pkg/buildinfo"
	"github.com/containership/cluster-manager/pkg/env"
)

type containershipClusterMetadata struct {
	ClusterID      string         `json:"cluster_id"`
	OrganizationID string         `json:"organization_id"`
	BuildInfo      buildinfo.Spec `json:"build_info"`
}

type metadata struct {
	Containership containershipClusterMetadata `json:"containership"`
	Timestamp     time.Time                    `json:"timestamp"`
}

// Get returns Containership and node metadata
func (meta *Metadata) Get(w http.ResponseWriter, r *http.Request) {
	m := &metadata{
		Containership: containershipClusterMetadata{
			ClusterID:      env.ClusterID(),
			OrganizationID: env.OrganizationID(),
			BuildInfo:      buildinfo.Get(),
		},
		Timestamp: time.Now().UTC(),
	}

	respondWithJSON(w, http.StatusOK, m)
}
