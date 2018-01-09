package server

import (
	"github.com/containership/cloud-agent/internal/server/handlers"
)

// Mux example to get url variables
// a.Router.HandleFunc("/product/{id:[0-9]+}", m.Get).Methods("GET")
//     vars := mux.Vars(r)
//     id, err := strconv.Atoi(vars["id"])
//
func (a *CSServer) initializeRoutes() {
	m := &handlers.Metadata{}

	a.Router.HandleFunc("/metadata", m.Get).Methods("GET")
}
