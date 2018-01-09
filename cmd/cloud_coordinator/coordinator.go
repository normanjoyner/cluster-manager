package main

import (
	"time"

	"github.com/containership/cloud-agent/internal/coordinator"
	"github.com/containership/cloud-agent/internal/k8sutil"
	"github.com/containership/cloud-agent/internal/log"
)

func main() {
	log.Info("Starting Containership coordinator...")

	stopCh := make(chan struct{})

	kubeInformerFactory := k8sutil.API().NewKubeSharedInformerFactory(time.Second * 10)
	csInformerFactory := k8sutil.CSAPI().NewCSSharedInformerFactory(time.Second * 10)

	controller := coordinator.NewController(
		k8sutil.API().Client(), k8sutil.CSAPI().Client(), kubeInformerFactory, csInformerFactory)

	go kubeInformerFactory.Start(stopCh)
	go csInformerFactory.Start(stopCh)

	if err := controller.Run(2, stopCh); err != nil {
		log.Fatal("Error running controller:", err.Error())
	}
}
