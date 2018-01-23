package handlers

import (
	"net/http"
	"time"

	"github.com/containership/cloud-agent/internal/buildinfo"
	"github.com/containership/cloud-agent/internal/envvars"
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
			ClusterID:      envvars.GetClusterID(),
			OrganizationID: envvars.GetOrganizationID(),
			BuildInfo:      buildinfo.Get(),
		},
		Timestamp: time.Now(),
	}

	respondWithJSON(w, http.StatusOK, m)
}
