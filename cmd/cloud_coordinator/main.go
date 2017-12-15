package main

import (
	"log"
	"runtime"
	"time"

	"github.com/containership/cloud-agent/internal/envvars"
	"github.com/containership/cloud-agent/internal/resources"
)

func main() {
	log.Println("Starting containership coordinator...")

	watchResources()
	syncResources()

	runtime.Goexit()
}

func watchResources() {
	log.Println("Watching resources...")

	ticker := time.NewTicker(time.Duration(envvars.GetAgentSyncIntervalInSeconds()) * time.Second)
	quit := make(chan struct{})

	go func() {
		for {
			select {
			case <-ticker.C:
				resources.Loadbalancers.Reconcile()
				resources.RoleBasedAccessControls.Reconcile()
				resources.Registries.Reconcile()
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}

func syncResources() {
	log.Println("Syncing resources...")

	ticker := time.NewTicker(time.Duration(envvars.GetAgentSyncIntervalInSeconds()) * time.Second)
	quit := make(chan struct{})

	go func() {
		for {
			select {
			case <-ticker.C:
				resources.Loadbalancers.Sync(resources.Loadbalancers.Write)
				resources.RoleBasedAccessControls.Sync(resources.RoleBasedAccessControls.Write)
				resources.Registries.Sync(resources.Registries.Write)

				// Host level resources, need to let all hosts know to sync
				resources.SSHKeys.Sync(updateAgents)
				resources.Firewalls.Sync(updateAgents)
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}

func updateAgents() {
	log.Println("Make call to each agent pod..")
	// TODO
	// get all agents endpoints
	// make request to /update
}
