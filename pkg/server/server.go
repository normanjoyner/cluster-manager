package server

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/containership/cloud-agent/pkg/env"
	"github.com/containership/cloud-agent/pkg/log"
)

// CSServer defines the server
type CSServer struct {
	router *mux.Router
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
	port := env.CSServerPort()

	cs.run(fmt.Sprintf(":%s", port))
}

func (cs *CSServer) initialize() {
	cs.router = mux.NewRouter()
	cs.initializeRoutes()
}

func (cs *CSServer) run(addr string) {
	log.Fatal(http.ListenAndServe(addr, cs.router))
}
