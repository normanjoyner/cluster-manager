package handlers

import (
	"net/http"

	"github.com/containership/cloud-agent/internal/coordinator"
	"github.com/containership/cloud-agent/internal/log"
)

// Post stops cloud synchronization and requests cleanup and termination
func (terminate *Terminate) Post(w http.ResponseWriter, r *http.Request) {
	log.Info("Terminate request received")

	coordinator.RequestTerminate()

	respondWithStatus(w, http.StatusAccepted)
}
