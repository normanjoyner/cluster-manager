package main

import (
	"flag"
	"log"
	"time"

	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/containership/cloud-agent/internal/coordinator"
	"github.com/containership/cloud-agent/internal/k8sutil"
	csclientset "github.com/containership/cloud-agent/pkg/client/clientset/versioned"
	csinformers "github.com/containership/cloud-agent/pkg/client/informers/externalversions"
)

func main() {
	log.Println("Starting Containership coordinator...")

	flag.Parse()

	stopCh := make(chan struct{})

	// TODO need to kick off a sync() routine as well for cs->k8s sync

	// TODO use k8sutil
	/*
		log.Println("Using in cluster k8s config")
		config, err := rest.InClusterConfig()

		if err != nil {
			log.Fatalf("rest.InClusterConfig() failed: %s", err.Error())
		}

		kubeClient, err := kubernetes.NewForConfig(config)

		if err != nil {
			log.Fatalf("kubernetes.NewForConfig() failed: %s", err.Error())
		}
	*/

	kubeClient := k8sutil.Client()

	csClient, err := csclientset.NewForConfig(config)
	if err != nil {
		log.Fatalf("csclientset.NewForConfig() failed: %s", err.Error())
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*10)
	csInformerFactory := csinformers.NewSharedInformerFactory(csClient, time.Second*10)

	controller := coordinator.NewController(
		kubeClient, csClient, kubeInformerFactory, csInformerFactory)

	go kubeInformerFactory.Start(stopCh)
	go csInformerFactory.Start(stopCh)

	if err = controller.Run(2, stopCh); err != nil {
		log.Fatalf("Error running controller: %s", err.Error())
	}
}
