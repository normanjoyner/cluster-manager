package handlers

import (
	"net/http"

	"github.com/containership/cloud-agent/internal/resources"
)

// POST forces host to sync
func (s *Sync) POST(w http.ResponseWriter, r *http.Request) {
	resources.Firewalls.Sync(resources.Firewalls.Write)
	resources.SSHKeys.Sync(resources.SSHKeys.Write)

	respondWithStatus(w, http.StatusOK)
}
