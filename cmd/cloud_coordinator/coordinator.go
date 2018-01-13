package main

import (
	"flag"
	"runtime"

	"github.com/containership/cloud-agent/internal/coordinator"
	"github.com/containership/cloud-agent/internal/log"
	"github.com/containership/cloud-agent/internal/server"
)

func main() {
	log.Info("Starting Containership coordinator...")

	// We don't have any of our own flags to parse, but k8s packages want to
	// use glog and we have to pass flags to that to configure it to behave
	// in a sane way.
	flag.Parse()

	coordinator.Initialize()
	go coordinator.Run()

	// Run the http server
	s := server.New()
	go s.Run()

	runtime.Goexit()
}
