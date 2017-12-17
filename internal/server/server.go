package server

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/containership/cloud-agent/internal/envvars"
)

// CSServer defines the server
type CSServer struct {
	Router *mux.Router
}

// New creates a new server
func New() *CSServer {
	cs := &CSServer{}
	cs.initialize()

	return cs
}

// Run is exported for main agent to start server,
// which is what containership uses to talk with the cluster
func (cs *CSServer) Run() {
	port := envvars.GetCSServerPort()

	cs.run(fmt.Sprintf(":%s", port))
}

func (cs *CSServer) initialize() {
	cs.Router = mux.NewRouter()
	cs.initializeRoutes()
}

func (cs *CSServer) run(addr string) {
	log.Fatal(http.ListenAndServe(addr, cs.Router))
}
