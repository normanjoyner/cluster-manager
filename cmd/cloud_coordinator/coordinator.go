package main

import (
	"flag"
	"runtime"

	"github.com/containership/cluster-manager/pkg/buildinfo"
	"github.com/containership/cluster-manager/pkg/coordinator"
	"github.com/containership/cluster-manager/pkg/env"
	"github.com/containership/cluster-manager/pkg/log"
	"github.com/containership/cluster-manager/pkg/server"
)

func main() {
	log.Info("Starting Containership Cloud Coordinator...")
	log.Infof("Version: %s", buildinfo.String())
	log.Infof("Go Version: %s", runtime.Version())

	// We don't have any of our own flags to parse, but k8s packages want to
	// use glog and we have to pass flags to that to configure it to behave
	// in a sane way.
	flag.Set("logtostderr", "true")
	flag.Parse()

	env.Dump()

	coordinator.Initialize()
	go coordinator.Run()

	// Run the http server
	s := server.New()
	go s.Run()

	runtime.Goexit()
}
