package main

import (
	"flag"
	"runtime"

	"github.com/containership/cloud-agent/internal/agent"
	"github.com/containership/cloud-agent/internal/buildinfo"
	"github.com/containership/cloud-agent/internal/env"
	"github.com/containership/cloud-agent/internal/log"
)

func main() {
	log.Info("Starting Containership Cloud Agent...")
	log.Infof("Version: %s", buildinfo.String())
	log.Infof("Go Version: %s", runtime.Version())

	// We don't have any of our own flags to parse, but k8s packages want to
	// use glog and we have to pass flags to that to configure it to behave
	// in a sane way.
	flag.Parse()

	env.Dump()

	agent.Initialize()
	go agent.Run()

	// Note that we'll never actually exit because some goroutines out of our
	// control (e.g. the glog flush daemon) will continue to run).
	runtime.Goexit()
}
