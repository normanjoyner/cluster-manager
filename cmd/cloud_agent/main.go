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
	watchResources()

	s := server.New()
	s.Run()
}

func watchResources() {
	log.Println("Watching resources...")

	ticker := time.NewTicker(time.Duration(envvars.GetAgentSyncIntervalInSeconds()) * time.Second)
	quit := make(chan struct{})

	go func() {
		for {
			select {
			case <-ticker.C:
				resources.SSHKeys.Reconcile()
				resources.Firewalls.Reconcile()
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}
