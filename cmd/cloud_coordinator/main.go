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

	// Cluster level
	loadbalancers := resources.NewLoadbalancers()
	rbacs := resources.NewRoleBasedAccessControls()
	registries := resources.NewRegistries()

	// Host level
	sshKeys := resources.NewSSHKeys()
	firewalls := resources.NewFirewalls()

	resources.Register(loadbalancers, loadbalancers.Write)
	resources.Register(rbacs, rbacs.Write)
	resources.Register(registries, registries.Write)
	resources.Register(sshKeys, updateAgents)
	resources.Register(firewalls, updateAgents)

	// Kick off watch loop
	watchResources()

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
				// compare disk to cache, if diff then invalidate cache
				resources.ReconcileByType(resources.ResourceTypeCluster)
				// hit api to grab latest, if diff from cache then update cache and call passed func
				resources.Sync()
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}

func updateAgents() {
	log.Println("Make call to each agent pod...")
	// TODO
	// get all agents endpoints
	// make request to /update
}
