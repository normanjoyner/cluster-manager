package server

import (
	"github.com/containership/cloud-agent/internal/server/handlers"
)

// initializeRoutes sets up all routes
func (a *CSServer) initializeRoutes() {
	m := &handlers.Metadata{}
	c := &handlers.Terminate{}

	a.Router.HandleFunc("/metadata", m.Get).Methods("GET")
	a.Router.HandleFunc("/terminate", c.Post).Methods("POST")
}
