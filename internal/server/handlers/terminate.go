package handlers

import (
	"net/http"

	"github.com/containership/cloud-agent/internal/coordinator"
	"github.com/containership/cloud-agent/internal/log"
)

// Delete stops cloud synchronization and requests cleanup and termination
func (terminate *Terminate) Delete(w http.ResponseWriter, r *http.Request) {
	log.Info("Terminate request received")

	coordinator.RequestTerminate()

	respondWithStatus(w, http.StatusAccepted)
}
