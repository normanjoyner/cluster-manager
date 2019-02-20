package main

import (
	"flag"
	"runtime"

	"github.com/containership/cluster-manager/pkg/agent"
	"github.com/containership/cluster-manager/pkg/buildinfo"
	"github.com/containership/cluster-manager/pkg/env"
	"github.com/containership/cluster-manager/pkg/log"
)

func main() {
	log.Info("Starting Containership Cloud Agent...")
	log.Infof("Version: %s", buildinfo.String())
	log.Infof("Go Version: %s", runtime.Version())

	// We don't have any of our own flags to parse, but k8s packages want to
	// use glog and we have to pass flags to that to configure it to behave
	// in a sane way.
	flag.Set("logtostderr", "true")
	flag.Parse()

	env.Dump()

	agent.Initialize()
	go agent.Run()

	// Note that we'll never actually exit because some goroutines out of our
	// control (e.g. the glog flush daemon) will continue to run).
	runtime.Goexit()
}
