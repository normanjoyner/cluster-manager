package main

import (
	"flag"
	"runtime"
	"time"

	"github.com/containership/cloud-agent/internal/agent"
	"github.com/containership/cloud-agent/internal/k8sutil"
	"github.com/containership/cloud-agent/internal/log"
	"github.com/containership/cloud-agent/internal/resources/sysuser"
)

func main() {
	log.Info("Starting Containership agent...")

	// We don't have any of our own flags to parse, but k8s packages want to
	// use glog and we have to pass flags to that to configure it to behave
	// in a sane way.
	flag.Parse()

	// Failure to initialize what we need for SSH to work is not a fatal error
	// because the user may not have performed the manual steps required to
	// make the feature work (also, we may just be running in a dev
	// environment).
	if err := sysuser.InitializeAuthorizedKeysFileStructure(); err != nil {
		log.Info("Could not initialize authorized_keys:", err.Error())
		log.Info("SSH feature will not work without manual intervention")
	}

	csInformerFactory := k8sutil.CSAPI().NewCSSharedInformerFactory(time.Second * 10)

	// TODO change NewController to allow for different types (we need
	// Firewalls too eventually)
	controller := agent.NewController(k8sutil.CSAPI().Client(), csInformerFactory)

	// Kick off the informer factory
	stopCh := make(chan struct{})
	csInformerFactory.Start(stopCh)

	// Run controller until error
	// Each controller is pretty lightweight so one worker should be fine
	if err := controller.Run(1, stopCh); err != nil {
		log.Fatal("Error running controller:", err.Error())
	}

	runtime.Goexit()
}
