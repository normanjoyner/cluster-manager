package main

import (
	"flag"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/containership/cloud-agent/internal/agent"
	"github.com/containership/cloud-agent/internal/buildinfo"
	"github.com/containership/cloud-agent/internal/env"
	"github.com/containership/cloud-agent/internal/k8sutil"
	"github.com/containership/cloud-agent/internal/log"
	"github.com/containership/cloud-agent/internal/resources/sysuser"
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

	// Failure to initialize what we need for SSH to work is not a fatal error
	// because the user may not have performed the manual steps required to
	// make the feature work (also, we may just be running in a dev
	// environment).
	if err := sysuser.InitializeAuthorizedKeysFileStructure(); err != nil {
		log.Info("Could not initialize authorized_keys:", err.Error())
		log.Info("SSH feature will not work without manual intervention")
	}

	interval := env.AgentInformerSyncInterval()
	csInformerFactory := k8sutil.CSAPI().NewCSSharedInformerFactory(interval)

	// TODO change NewController to allow for different types (we need
	// Firewalls too eventually)
	controller := agent.NewController(k8sutil.CSAPI().Client(), csInformerFactory)

	// Kick off the informer factory
	stopCh := make(chan struct{})
	csInformerFactory.Start(stopCh)

	// SIGTERM is sent when a pod is deleted in Kubernetes. The agent needs to
	// clean up host-level resources within the grace period before the
	// follow-up SIGKILL arrives (default grace period being 30s).
	signals := make(chan os.Signal)
	signal.Notify(signals, syscall.SIGTERM)
	go signalHandler(signals, stopCh)

	// Run controller until error or requested to stop
	// Each controller is pretty lightweight so one worker should be fine
	if err := controller.Run(1, stopCh); err != nil {
		log.Fatal("Error running controller: ", err.Error())
	}

	// Note that we'll never actually exit because some goroutines out of our
	// control (e.g. the glog flush daemon) will continue to run).
	runtime.Goexit()
}

func signalHandler(signals chan os.Signal, stopCh chan struct{}) {
	for {
		sig := <-signals
		switch sig {
		case syscall.SIGTERM:
			log.Infof("SIGTERM received - attempting to shut down gracefully")
			close(stopCh)
			signal.Stop(signals)
			return
		default:
			// It should be impossible to get here since we're only listening
			// on SIGTERM, but let's log it just for fun
			log.Infof("Signal %v received", sig)
		}
	}
}
