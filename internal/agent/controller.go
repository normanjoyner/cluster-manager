package agent

import (
	"fmt"
	"time"

	"github.com/fsnotify/fsnotify"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/containership/cloud-agent/internal/constants"
	"github.com/containership/cloud-agent/internal/envvars"
	"github.com/containership/cloud-agent/internal/log"
	"github.com/containership/cloud-agent/internal/resources/sysuser"
	containershipv3 "github.com/containership/cloud-agent/pkg/apis/containership.io/v3"
	csclientset "github.com/containership/cloud-agent/pkg/client/clientset/versioned"
	csinformers "github.com/containership/cloud-agent/pkg/client/informers/externalversions"
	cslisters "github.com/containership/cloud-agent/pkg/client/listers/containership.io/v3"
)

const (
	// Commands to watcher to control it
	fileWatchStop = iota
	fileWatchStart
	// Command from watcher indicating command processing completion
	fileWatchCmdComplete
)

// Controller is the agent controller which watches for CRD changes and reports
// back when a write to host is needed
// TODO this needs to be able to be instantiated with a different resource type
type Controller struct {
	clientset   csclientset.Interface
	usersLister cslisters.UserLister
	usersSynced cache.InformerSynced
	workqueue   workqueue.RateLimitingInterface

	requestWriteCh chan bool
	fileWatchCmdCh chan int
}

// NewController creates a new agent Controller
func NewController(
	clientset csclientset.Interface,
	csInformerFactory csinformers.SharedInformerFactory) *Controller {

	// Create an informer from the factory so that we share the underlying
	// cache with other controllers
	userInformer := csInformerFactory.Containership().V3().Users()

	controller := &Controller{
		clientset:      clientset,
		usersLister:    userInformer.Lister(),
		usersSynced:    userInformer.Informer().HasSynced,
		workqueue:      workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Users"),
		requestWriteCh: make(chan bool),
		fileWatchCmdCh: make(chan int),
	}

	// All event handlers simply add to a workqueue to be processed by a worker
	userInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueUser,
		UpdateFunc: func(old, new interface{}) {
			newUser := new.(*containershipv3.User)
			oldUser := old.(*containershipv3.User)
			if oldUser.ResourceVersion == newUser.ResourceVersion {
				// This must be a syncInterval update - nothing has changed so
				// do nothing
				return
			}
			controller.enqueueUser(new)
		},
		DeleteFunc: controller.enqueueUser,
	})

	return controller
}

// Run kicks off the Controller with the given number of workers to process the
// workqueue
func (c *Controller) Run(numWorkers int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	log.Info("Starting User controller")

	log.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.usersSynced); !ok {
		return fmt.Errorf("Failed to wait for caches to sync")
	}

	log.Info("Starting write request handler")
	go c.handleWriteRequests(stopCh)

	go c.authorizedKeysWatcher()
	c.sendCmdToFileWatcher(fileWatchStart)

	log.Info("Starting workers")
	// Launch two workers to process User resources
	for i := 0; i < numWorkers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	log.Info("Started workers")
	<-stopCh
	log.Info("Shutting down workers")

	return nil
}

// runWorker continually requests that the next queue item be processed
func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem continually pops items off of the workqueue and handles
// them
func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workqueue.Forget(obj)
			runtime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// User resource to be synced.
		if err := c.syncHandler(key); err != nil {
			return fmt.Errorf("error syncing '%s': %s", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		log.Debugf("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		runtime.HandleError(err)
		return true
	}

	return true
}

// enqueueUser enqueues the key for a given user - it should not be called for
// other object types
func (c *Controller) enqueueUser(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.workqueue.AddRateLimited(key)
}

// syncHandler looks at the current state of the system and decides how to act.
// For the agent, this means just requesting a write since something
// interesting must have happened for us to get to this point. Note that write
// requests are rate limited on the other side, so we can request as many times
// as we want here and it will collapse to periodic writes of the latest cache
func (c *Controller) syncHandler(key string) error {
	log.Debugf("User updated: key=%s\n", key)
	c.requestWriteCh <- true
	return nil
}

// handleWriteRequests listens on a channel for write requests and periodically
// writes if there is any requests
// TODO add support for multiple resource types, either by modifying this or
// using a different structure
func (c *Controller) handleWriteRequests(stopCh <-chan struct{}) {
	interval := envvars.GetAgentSyncIntervalInSeconds()
	ticker := time.NewTicker(time.Duration(interval) * time.Second)

	writeRequested := false

	for {
		select {
		case <-ticker.C:
			if writeRequested {
				log.Debug("Writing resource to host...")
				err := c.writeAuthorizedUsers()
				if err == nil {
					writeRequested = false
				}
			}

		case <-c.requestWriteCh:
			log.Debug("Write requested")
			writeRequested = true

		case <-stopCh:
			log.Info("Write request handler stopping")
			ticker.Stop()
			return
		}
	}
}

func (c *Controller) writeAuthorizedUsers() error {
	// TODO for users this must filter on `ssh_access=true` once
	// RBAC is integrated on coordinator side (RBAC CRD Controller
	// adds the label)
	userPointers, err := c.usersLister.Users(constants.ContainershipNamespace).List(labels.NewSelector())
	if err != nil {
		return err
	}

	// Convert Users to UserSpecs
	users := make([]containershipv3.UserSpec, 0)
	for _, u := range userPointers {
		users = append(users, u.Spec)
	}

	log.Debugf("Users: %+v\n", users)

	// Stop file notifications while we write
	c.sendCmdToFileWatcher(fileWatchStop)
	log.Info("Writing authorized_keys")
	err = sysuser.WriteAuthorizedKeys(users)
	c.sendCmdToFileWatcher(fileWatchStart)

	return err
}

// sendCmdToFileWatcher sends a command to the file watcher routine and
// blocks until a command complete response is received
func (c *Controller) sendCmdToFileWatcher(cmd int) {
	c.fileWatchCmdCh <- cmd
	<-c.fileWatchCmdCh // wait for cmd complete
}

// authorizedKeysWatcher spins up an fsnotify watcher on the
// authorized_keys file and waits for events. If any events occur,
// we assume that there was an unauthorized change to the file and
// we send a write request on the write request channel. Note that
// when we write to the file ourselves, we must disable the file
// watcher by sending a Stop command and re-enable it with a Start
// command after writing.
func (c *Controller) authorizedKeysWatcher() {
	filename := sysuser.GetAuthorizedKeysFullPath()

	fileWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		// TODO consider attempting to handle this case
		log.Fatal(err)
	}
	defer fileWatcher.Close()

	for {
		select {
		case event := <-fileWatcher.Events:
			// We don't care what kind of event - always request a write
			log.Info("Unexpected authorized_key file event detected: ", event)
			c.requestWriteCh <- true

		case err := <-fileWatcher.Errors:
			log.Error("File watcher error:", err)

		case cmd := <-c.fileWatchCmdCh:
			switch cmd {
			case fileWatchStart:
				log.Debug("Starting file watcher")
				if err := fileWatcher.Add(filename); err != nil {
					// This should be benign and will resolve itself on next
					// authorized write
					log.Error("fileWatcher.Add() failed: ", err.Error())
				}
				c.fileWatchCmdCh <- fileWatchCmdComplete

			case fileWatchStop:
				log.Debug("Stopping file watcher")
				if err := fileWatcher.Remove(filename); err != nil {
					// This should be benign and will resolve itself on next
					// authorized write
					log.Error("fileWatcher.Remove() failed: ", err.Error())
				}
				c.fileWatchCmdCh <- fileWatchCmdComplete
			}
		}
	}
}
