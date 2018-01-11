package main

import (
	"runtime"

	"github.com/containership/cloud-agent/internal/coordinator"
	"github.com/containership/cloud-agent/internal/log"
	"github.com/containership/cloud-agent/internal/server"
)

func main() {
	log.Info("Starting Containership coordinator...")

	coordinator.Initialize()
	go coordinator.Run()

	// Run the http server
	s := server.New()
	go s.Run()

	runtime.Goexit()
}
