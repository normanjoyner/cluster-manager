package coordinator

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/spf13/afero"

	"github.com/containership/cloud-agent/internal/constants"
	"github.com/containership/cloud-agent/internal/k8sutil/kubectl"
	"github.com/containership/cloud-agent/internal/log"
	"github.com/containership/cloud-agent/internal/request"

	containershipv3 "github.com/containership/cloud-agent/pkg/apis/containership.io/v3"
	csclientset "github.com/containership/cloud-agent/pkg/client/clientset/versioned"
	csscheme "github.com/containership/cloud-agent/pkg/client/clientset/versioned/scheme"
	csinformers "github.com/containership/cloud-agent/pkg/client/informers/externalversions"
	cslisters "github.com/containership/cloud-agent/pkg/client/listers/containership.io/v3"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"

	corev1 "k8s.io/api/core/v1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

const (
	// Type of agent that runs this controller
	pluginControllerName = "plugin"

	pluginLabelKey = "containership.io/plugin-id"

	maxPluginControllerRetries = 5
)

// PluginController is the controller implementation for the containership
// setup
type PluginController struct {
	// kubeclientset is a standard kubernetes clientset
	clientset csclientset.Interface

	pluginLister  cslisters.PluginLister
	pluginsSynced cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder
}

// used for writing and deleting manifest files
var osFs = afero.NewOsFs()

// NewPluginController returns a new containership controller
func NewPluginController(kubeclientset kubernetes.Interface, clientset csclientset.Interface, csInformerFactory csinformers.SharedInformerFactory) *PluginController {
	log.Info(pluginControllerName, ": Creating event broadcaster")

	csscheme.AddToScheme(scheme.Scheme)
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(log.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})

	pc := &PluginController{
		clientset: clientset,
		workqueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Plugins"),
		recorder:  eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: pluginControllerName}),
	}

	// Instantiate resource informers
	pluginInformer := csInformerFactory.Containership().V3().Plugins()

	// namespace informer listens for add events an queues the namespace
	pluginInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    pc.enqueuePlugin,
		DeleteFunc: pc.enqueuePlugin,
	})

	// Listers are used for cache inspection and Synced functions
	// are used to wait for cache synchronization
	pc.pluginLister = pluginInformer.Lister()
	pc.pluginsSynced = pluginInformer.Informer().HasSynced

	log.Info(pluginControllerName, ": Setting up event handlers")

	return pc
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *PluginController) Run(numWorkers int, stopCh chan struct{}) {
	defer runtime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	log.Info(pluginControllerName, ": Starting controller")

	if ok := cache.WaitForCacheSync(
		stopCh,
		c.pluginsSynced); !ok {
		// If this channel is unable to wait for caches to sync we stop both
		// the containership controller, and the plugin controller
		close(stopCh)
		log.Error("failed to wait for caches to sync")
	}

	log.Info(pluginControllerName, ": Starting workers")
	// Launch numWorkers amount of workers to process resources
	for i := 0; i < numWorkers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	log.Info(pluginControllerName, ": Started workers")
	<-stopCh
	log.Info(pluginControllerName, ": Shutting down workers")
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *PluginController) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem continually pops items off of the workqueue and handles
// them
func (c *PluginController) processNextWorkItem() bool {
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
		// Plugin resource to be synced.
		err := c.pluginSyncHandler(key)
		return c.handleErr(err, key)
	}(obj)

	if err != nil {
		runtime.HandleError(err)
		return true
	}

	return true
}

func (c *PluginController) handleErr(err error, key interface{}) error {
	if err == nil {
		log.Debugf("Successfully synced '%s'", key)
		c.workqueue.Forget(key)
		return nil
	}

	if c.workqueue.NumRequeues(key) < maxPluginControllerRetries {
		c.workqueue.AddRateLimited(key)
		return fmt.Errorf("error syncing %q: %s. has been resynced %v times", key, err.Error(), c.workqueue.NumRequeues(key))
	}

	c.workqueue.Forget(key)
	log.Infof("Dropping Plugin %q out of the queue: %v", key, err)
	return err
}

// pluginSyncHandler uses kubectl to apply the plugin manifest if a plugin CRD
// exists, if the plugin CRD does not exist delete the associated plugin
// manifest files
func (c *PluginController) pluginSyncHandler(key string) error {
	_, name, _ := cache.SplitMetaNamespaceKey(key)
	plugin, err := c.pluginLister.Plugins(constants.ContainershipNamespace).Get(name)

	if err != nil {
		// If the plugin CRD does not exist delete the plugin manifest files
		if errors.IsNotFound(err) {
			return c.deletePlugin(name)
		}

		return err
	}

	// Create the plugin manifest files
	return c.applyPlugin(plugin)
}

// TODO: abstract out resources so we can pull in plugin url here
func makePluginManifestPath(ID string) string {
	return fmt.Sprintf("/organizations/{{.OrganizationID}}/clusters/{{.ClusterID}}/plugins/%s", ID)
}

func makeRequest(path string) ([]byte, error) {
	req, err := request.New(path, "GET", nil)
	if err != nil {
		return nil, err
	}

	resp, err := req.MakeRequest()
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func runAndLogFromKubectl(kc *kubectl.Kubectl) error {
	err := kc.Run()

	log.Debug("Kubectl output: ", string(kc.Output))
	if err != nil || len(kc.Error) != 0 {
		e := string(kc.Error)

		log.Error("Kubectl error: ", e)
		return fmt.Errorf("Kubectl error: %s", e)
	}

	return nil
}

func (c *PluginController) deletePlugin(name string) error {
	pluginPath := getPluginPath(name)

	var kc *kubectl.Kubectl
	if stat, err := os.Stat(pluginPath); os.IsNotExist(err) || !stat.IsDir() {
		// Covers edge case that if plugin does not have a directory with manifests
		// it will try and delete by the containership.io/plugin-id label
		label := pluginLabelKey + "=" + name
		kc = kubectl.NewDeleteByLabelCmd(constants.ContainershipNamespace, label)
	} else {
		kc = kubectl.NewDeleteCmd(getPluginPath(name))
	}

	err := runAndLogFromKubectl(kc)
	if err != nil {
		return err
	}

	// attempts to clean up the manifest directory that was created,
	// if there is an error it's ignored
	cleanUpPluginManifests(name)
	return nil
}

type jsonPluginsResponse struct {
	Manifests [][]interface{} `json:"manifests"`
}

// getAndFormatPlugin makes a request to get the plugin manifests from containership
// cloud, then takes them and splits them up to be written to files
// under the plugins/:plugin_id
func (c *PluginController) getAndFormatPlugin(ID string) error {
	path := makePluginManifestPath(ID)
	bytes, err := makeRequest(path)
	if err != nil {
		return err
	}

	var plugin jsonPluginsResponse
	err = json.Unmarshal(bytes, &plugin)
	if err != nil {
		return err
	}

	for index, resources := range plugin.Manifests {
		manifestBundle := make([]byte, 0)
		for _, r := range resources {
			// Re-Marshal the generic object so we can have each
			// as []byte
			unstructuredBytes, _ := json.Marshal(r)
			manifestBundle = append(manifestBundle, unstructuredBytes...)
		}

		err = writePluginManifests(ID, index, manifestBundle)
		// we only want to short circuit and return if there was an error
		// else continue processing the manifests
		if err != nil {
			return err
		}
	}

	return nil
}

func getPluginPath(id string) string {
	return path.Join("/plugins", id)
}

func getPluginFilename(id string, index int) string {
	return path.Join(getPluginPath(id), fmt.Sprintf("%d.json", index))
}

// writePluginManifests takes the manifests and writes them to a .json file
// writing them named as their index since kubectl apply runs files in
// alphanumeric order
func writePluginManifests(name string, index int, manifest []byte) error {
	path := getPluginPath(name)

	err := osFs.MkdirAll(path, os.ModePerm)
	if err != nil {
		return err
	}

	filename := getPluginFilename(name, index)

	return ioutil.WriteFile(filename, manifest, 0644)
}

func cleanUpPluginManifests(name string) {
	err := osFs.RemoveAll(getPluginPath(name))
	if err != nil {
		log.Error(err)
	}
}

// applyPlugin takes the manifests under the plugins id and runs them through
// kubectl apply
func (c *PluginController) applyPlugin(plugin *containershipv3.Plugin) error {
	err := c.getAndFormatPlugin(plugin.Spec.ID)
	if err != nil {
		return err
	}

	kc := kubectl.NewApplyCmd(getPluginPath(plugin.Spec.ID))

	err = kc.Run()
	c.recorder.Event(plugin, corev1.EventTypeNormal, "Apply", string(kc.Output))
	if err != nil || len(kc.Error) != 0 {
		e := string(kc.Error)

		c.recorder.Event(plugin, corev1.EventTypeWarning, "ApplyError", e)
		return fmt.Errorf("Error creating plugin: %s", e)
	}

	return nil
}

// enqueuePlugin enqueues the plugin on add or delete
func (c *PluginController) enqueuePlugin(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		runtime.HandleError(err)
		return
	}

	c.workqueue.AddRateLimited(key)
}
