package coordinator

import (
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/scheme"

	csscheme "github.com/containership/cluster-manager/pkg/client/clientset/versioned/scheme"
	csinformers "github.com/containership/cluster-manager/pkg/client/informers/externalversions"
	"github.com/containership/cluster-manager/pkg/env"
	"github.com/containership/cluster-manager/pkg/k8sutil"
	"github.com/containership/cluster-manager/pkg/log"

	cerebralscheme "github.com/containership/cerebral/pkg/client/clientset/versioned/scheme"
	cerebralinformers "github.com/containership/cerebral/pkg/client/informers/externalversions"
)

var (
	kubeInformerFactory     kubeinformers.SharedInformerFactory
	csInformerFactory       csinformers.SharedInformerFactory
	cerebralInformerFactory cerebralinformers.SharedInformerFactory
	regController           *RegistryController
	csController            *ContainershipController
	plgnController          *PluginController
	cupController           *UpgradeController
	cloudSynchronizer       *CloudSynchronizer
)

// Initialize creates the informer factories, controller, and synchronizer.
func Initialize() {
	if err := k8sutil.Initialize(); err != nil {
		log.Fatal("Could not initialize k8sutil: ", err)
	}

	// Register our scheme with main k8s scheme so we can forward custom events
	// properly
	csscheme.AddToScheme(scheme.Scheme)
	// Register the cerebral scheme with main k8s scheme so we can forward custom events
	// properly
	cerebralscheme.AddToScheme(scheme.Scheme)

	// Create Informer factories. All Informers should be created from these
	// factories in order to share the same underlying caches.
	interval := env.CoordinatorInformerSyncInterval()
	kubeInformerFactory = k8sutil.API().NewKubeSharedInformerFactory(interval)
	csInformerFactory = k8sutil.CSAPI().NewCSSharedInformerFactory(interval)
	cerebralInformerFactory = k8sutil.CerebralAPI().NewCerebralSharedInformerFactory(interval)

	regController = NewRegistryController(
		k8sutil.API().Client(), k8sutil.CSAPI().Client(), kubeInformerFactory, csInformerFactory)

	csController = NewContainershipController(
		k8sutil.API().Client(), kubeInformerFactory)

	plgnController = NewPluginController(
		k8sutil.API().Client(), k8sutil.CSAPI().Client(), csInformerFactory)

	if env.IsClusterUpgradeEnabled() {
		cupController = NewUpgradeController(
			k8sutil.API().Client(), k8sutil.CSAPI().Client(), kubeInformerFactory, csInformerFactory)
	}

	// Synchronizer needs to be created before any jobs start so
	// that all needed index functions can be added to the
	// informers
	cloudSynchronizer = NewCloudSynchronizer(csInformerFactory, cerebralInformerFactory)
}

// Run kicks off the informer factories, controller, and synchronizer.
func Run() {
	// Kick off the informer factories
	stopCh := make(chan struct{})
	kubeInformerFactory.Start(stopCh)
	csInformerFactory.Start(stopCh)
	cerebralInformerFactory.Start(stopCh)

	cloudSynchronizer.Run()

	go csController.Run(1, stopCh)
	go plgnController.Run(1, stopCh)
	go regController.Run(1, stopCh)

	if env.IsClusterUpgradeEnabled() {
		go cupController.Run(1, stopCh)
	}

	// if stopCh is closed something went wrong
	<-stopCh
	log.Fatal("There was an error while running the coordinator's controllers")
}

// RequestTerminate requests to stop syncing, clean up, and terminate
func RequestTerminate() {
	cloudSynchronizer.RequestTerminate()
}
