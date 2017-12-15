package server

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/containership/cloud-agent/internal/envvars"
)

type csServer struct {
	Router *mux.Router
}

func New() *csServer {
	cs := &csServer{}
	cs.initialize()

	return cs
}

// Run is exported for main agent to start server,
// which is what containership uses to talk with the cluster
func (cs *csServer) Run() {
	port := envvars.GetCSServerPort()

	cs.run(fmt.Sprintf(":%s", port))
}

func (cs *csServer) initialize() {
	cs.Router = mux.NewRouter()
	cs.initializeRoutes()
}

func (cs *csServer) run(addr string) {
	log.Fatal(http.ListenAndServe(addr, cs.Router))
}
