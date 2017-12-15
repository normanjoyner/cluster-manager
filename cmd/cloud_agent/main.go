package main

import (
	"log"
	"time"

	"github.com/containership/cloud-agent/internal/envvars"
	"github.com/containership/cloud-agent/internal/resources"
	"github.com/containership/cloud-agent/internal/server"
)

func main() {
	log.Println("Starting containership agent...")

	// The agent only cares about host-level resources
	sshKeys := resources.NewSSHKeys()
	firewalls := resources.NewFirewalls()

	resources.Register(sshKeys, sshKeys.Write)
	resources.Register(firewalls, firewalls.Write)

	// Kick off the reconciliation loop
	watchResources()

	// Run the http server
	s := server.New()
	s.Run()
}

func watchResources() {
	ticker := time.NewTicker(time.Duration(envvars.GetAgentSyncIntervalInSeconds()) * time.Second)
	quit := make(chan struct{})

	go func() {
		for {
			select {
			case <-ticker.C:
				resources.ReconcileByType(resources.ResourceTypeHost)
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}
