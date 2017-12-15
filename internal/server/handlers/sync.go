package handlers

import (
	"net/http"

	"github.com/containership/cloud-agent/internal/resources"
)

// POST forces host to sync
func (s *Sync) POST(w http.ResponseWriter, r *http.Request) {
	// TODO take path/query param(s) to indicate which resources to sync
	// TODO should be able to identify which particular sync failed if multiple
	err := resources.Sync()

	if err == nil {
		respondWithStatus(w, http.StatusOK)
	} else {
		respondWithStatus(w, http.StatusInternalServerError)
	}
}
