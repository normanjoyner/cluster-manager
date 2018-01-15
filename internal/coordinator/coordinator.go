package coordinator

import (
	"time"

	kubeinformers "k8s.io/client-go/informers"

	"github.com/containership/cloud-agent/internal/k8sutil"
	"github.com/containership/cloud-agent/internal/log"
	csinformers "github.com/containership/cloud-agent/pkg/client/informers/externalversions"
)

var (
	kubeInformerFactory kubeinformers.SharedInformerFactory
	csInformerFactory   csinformers.SharedInformerFactory
	regController       *RegistryController
	csController        *ContainershipController
	cloudSynchronizer   *CloudSynchronizer
)

// Initialize creates the informer factories, controller, and synchronizer.
func Initialize() {
	// Create Informer factories. All Informers should be created from these
	// factories in order to share the same underlying caches.
	kubeInformerFactory = k8sutil.API().NewKubeSharedInformerFactory(time.Second * 10)
	csInformerFactory = k8sutil.CSAPI().NewCSSharedInformerFactory(time.Second * 10)

	regController = NewRegistryController(
		k8sutil.API().Client(), k8sutil.CSAPI().Client(), kubeInformerFactory, csInformerFactory)

	csController = NewContainershipController(
		k8sutil.API().Client(), kubeInformerFactory)

	// Synchronizer needs to be created before any jobs start so
	// that all needed index functions can be added to the
	// informers
	cloudSynchronizer = NewCloudSynchronizer(csInformerFactory)
}

// Run kicks off the informer factories, controller, and synchronizer.
func Run() {
	// Kick off the informer factories
	stopCh := make(chan struct{})
	kubeInformerFactory.Start(stopCh)
	csInformerFactory.Start(stopCh)

	cloudSynchronizer.Run()

	go regController.Run(2, stopCh)
	go csController.Run(2, stopCh)

	// if stopCh is closed something went wrong
	<-stopCh
	log.Fatal("There was an error while running the coordinator's controllers")
}

// RequestTerminate requests to stop syncing, clean up, and terminate
func RequestTerminate() {
	cloudSynchronizer.RequestTerminate()
}
