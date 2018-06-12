package handlers

import (
	"net/http"

	"github.com/containership/cloud-agent/pkg/coordinator"
	"github.com/containership/cloud-agent/pkg/log"
)

// Delete stops cloud synchronization and requests cleanup and termination
func (terminate *Terminate) Delete(w http.ResponseWriter, r *http.Request) {
	log.Info("Terminate request received")

	coordinator.RequestTerminate()

	respondWithStatus(w, http.StatusAccepted)
}
