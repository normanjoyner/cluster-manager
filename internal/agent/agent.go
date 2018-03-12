package agent

import (
	"os"
	"os/signal"
	"syscall"

	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/containership/cloud-agent/internal/env"
	"github.com/containership/cloud-agent/internal/k8sutil"
	"github.com/containership/cloud-agent/internal/log"
	"github.com/containership/cloud-agent/internal/resources/sysuser"
	csscheme "github.com/containership/cloud-agent/pkg/client/clientset/versioned/scheme"
	csinformers "github.com/containership/cloud-agent/pkg/client/informers/externalversions"
)

var (
	kubeInformerFactory kubeinformers.SharedInformerFactory
	csInformerFactory   csinformers.SharedInformerFactory
	userController      *UserController
	cupController       *UpgradeController
)

// Initialize creates the informer factories and controllers.
func Initialize() {
	// Register our scheme with main k8s scheme so we can forward custom events
	// properly
	csscheme.AddToScheme(scheme.Scheme)

	// Failure to initialize what we need for SSH to work is not a fatal error
	// because the user may not have performed the manual steps required to
	// make the feature work (also, we may just be running in a dev
	// environment).
	if err := sysuser.InitializeAuthorizedKeysFileStructure(); err != nil {
		log.Info("Could not initialize authorized_keys:", err.Error())
		log.Info("SSH feature will not work without manual intervention")
	}

	// Create Informer factories. All Informers should be created from these
	// factories in order to share the same underlying caches.
	interval := env.AgentInformerSyncInterval()
	kubeInformerFactory = k8sutil.API().NewKubeSharedInformerFactory(interval)
	csInformerFactory = k8sutil.CSAPI().NewCSSharedInformerFactory(interval)

	userController = NewUserController(
		k8sutil.CSAPI().Client(), csInformerFactory)

	cupController = NewUpgradeController(
		k8sutil.CSAPI().Client(), csInformerFactory, k8sutil.API().Client(), kubeInformerFactory)
}

// Run kicks off the informer factories and controller.
func Run() {
	// Kick off the informer factories
	stopCh := make(chan struct{})

	// SIGTERM is sent when a pod is deleted in Kubernetes. The agent needs to
	// clean up host-level resources within the grace period before the
	// follow-up SIGKILL arrives (default grace period being 30s).
	signals := make(chan os.Signal)
	signal.Notify(signals, syscall.SIGTERM)
	go signalHandler(signals, stopCh)

	kubeInformerFactory.Start(stopCh)
	csInformerFactory.Start(stopCh)

	go userController.Run(1, stopCh)
	go cupController.Run(1, stopCh)

	// if stopCh is closed something went wrong
	<-stopCh
	log.Fatal("There was an error while running the agent's controllers")
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
