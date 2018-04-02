package coordinator

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	"github.com/spf13/afero"

	"github.com/containership/cloud-agent/internal/constants"
	"github.com/containership/cloud-agent/internal/k8sutil/kubectl"
	"github.com/containership/cloud-agent/internal/log"
	"github.com/containership/cloud-agent/internal/request"
	"github.com/containership/cloud-agent/internal/tools"

	containershipv3 "github.com/containership/cloud-agent/pkg/apis/containership.io/v3"
	csclientset "github.com/containership/cloud-agent/pkg/client/clientset/versioned"
	csinformers "github.com/containership/cloud-agent/pkg/client/informers/externalversions"
	cslisters "github.com/containership/cloud-agent/pkg/client/listers/containership.io/v3"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	corev1 "k8s.io/api/core/v1"
)

const (
	// Type of agent that runs this controller
	pluginControllerName = "PluginController"

	pluginLabelKey = "containership.io/plugin-id"

	pluginDelayBetweenRetries = 30 * time.Second

	maxPluginControllerRetries = 10
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
	rateLimiter := workqueue.NewItemExponentialFailureRateLimiter(pluginDelayBetweenRetries, pluginDelayBetweenRetries)

	pc := &PluginController{
		clientset: clientset,
		workqueue: workqueue.NewNamedRateLimitingQueue(rateLimiter, "Plugin"),
		recorder:  tools.CreateAndStartRecorder(kubeclientset, pluginControllerName),
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
			log.Errorf("expected string in workqueue but got %#v", obj)
			return nil
		}

		// Run the syncHandler, passing it the namespace/name string of the
		// Plugin resource to be synced.
		err := c.pluginSyncHandler(key)
		return c.handleErr(err, key)
	}(obj)

	if err != nil {
		log.Error(err)
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
		return fmt.Errorf("error syncing %q: %s. Has been resynced %v times", key, err.Error(), c.workqueue.NumRequeues(key))
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

	log.Debugf("%s syncing plugin of type %q with implementation %q", pluginControllerName, plugin.Spec.Type, plugin.Spec.Implementation)
	// Create the plugin manifest files
	return c.applyPlugin(plugin)
}

// TODO: abstract out resources so we can pull in plugin url here
func makePluginManifestPath(ID string) string {
	return fmt.Sprintf("/organizations/{{.OrganizationID}}/clusters/{{.ClusterID}}/plugins/%s", ID)
}

func makeRequest(path string) ([]byte, error) {
	req, err := request.New(request.CloudServiceAPI, path, "GET", nil)
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

// deletePluginUsingFiles takes in a plugin id and the plugin path.
// It then looks at each file in the file directory and if it is a text file
// it will take the url that is written in the file, and delete using the url.
// If it is a json file it will use the file to delete the plugin.
func deletePluginUsingFiles(pluginID, pluginPath string) error {
	files, err := ioutil.ReadDir(pluginPath)
	if err != nil {
		return err
	}

	for _, f := range files {
		isTextFile := strings.Contains(f.Name(), ".txt")
		fullFilename := getPluginFilename(pluginID, f.Name())

		var kc *kubectl.Kubectl
		if !isTextFile {
			kc = kubectl.NewDeleteCmd(fullFilename)
		} else {
			b, err := ioutil.ReadFile(fullFilename)
			if err != nil {
				return err
			}

			url := string(b)
			kc = kubectl.NewDeleteCmd(url)
		}

		err := runAndLogFromKubectl(kc)
		if err != nil {
			return err
		}
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

		err := runAndLogFromKubectl(kc)
		if err != nil {
			return err
		}

		return nil
	}

	err := deletePluginUsingFiles(name, pluginPath)
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
	URLs      []string        `json:"urls"`
}

// getPlugin gets the plugin spec from cloud, or returns an error
func (c *PluginController) getPlugin(ID string) (*jsonPluginsResponse, error) {
	path := makePluginManifestPath(ID)
	bytes, err := makeRequest(path)
	if err != nil {
		return nil, err
	}

	var plugin jsonPluginsResponse
	err = json.Unmarshal(bytes, &plugin)
	if err != nil {
		return nil, err
	}

	return &plugin, nil
}

// formatPlugin takes the manifests from the plugin spec returned from cloud
// and splits them up to be written to files under the plugins/:plugin_id. It
// then returns an array of strings, which are the file paths.
func formatPlugin(spec containershipv3.PluginSpec, pluginDetails *jsonPluginsResponse) ([]string, error) {
	manifests := make([]string, 0)

	for index, resources := range pluginDetails.Manifests {
		manifestBundle := make([]byte, 0)
		for _, r := range resources {
			// Re-Marshal the generic object so we can have each
			// as []byte
			unstructuredBytes, _ := json.Marshal(r)
			manifestBundle = append(manifestBundle, unstructuredBytes...)
		}

		filename := getPluginFilename(spec.ID, buildPluginFilename(index, "json"))

		err := writePluginManifests(spec.ID, filename, manifestBundle)
		// we only want to short circuit and return if there was an error
		// else continue processing the manifests
		if err != nil {
			return nil, err
		}

		manifests = append(manifests, filename)
	}

	return manifests, nil
}

func getPluginPath(id string) string {
	return path.Join("/plugins", id)
}

func getPluginFilename(id string, name string) string {
	return path.Join(getPluginPath(id), name)
}

func buildPluginFilename(name int, fileType string) string {
	return fmt.Sprintf("%d.%s", name, fileType)
}

// writePluginManifests takes the manifests and writes them to a .json file.
// They are named as their index so that if `kubectl apply -f` is run on
// the directory they will be applied in the correct order, since kubectl apply
// runs files in alphanumeric order
func writePluginManifests(name string, filename string, manifest []byte) error {
	path := getPluginPath(name)

	err := osFs.MkdirAll(path, os.ModePerm)
	if err != nil {
		return err
	}

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
	pluginDetails, err := c.getPlugin(plugin.Spec.ID)
	if err != nil {
		return err
	}

	manifests := make([]string, 0)
	if pluginDetails.Manifests != nil {
		manifestFiles, err := formatPlugin(plugin.Spec, pluginDetails)

		if err != nil {
			return err
		}

		if len(manifestFiles) == 0 {
			// No manifests were created, so there's nothing to apply. This can
			// happen, for example, when the plugin implementation is k8s itself.
			return nil
		}

		manifests = manifestFiles
	}

	if pluginDetails.URLs != nil {
		for index, url := range pluginDetails.URLs {
			writePluginManifests(plugin.Spec.ID, getPluginFilename(plugin.Spec.ID, buildPluginFilename(index, "txt")), []byte(url))
			manifests = append(manifests, url)
		}
	}

	for _, manifest := range manifests {
		kc := kubectl.NewApplyCmd(manifest)

		err = kc.Run()
		c.recorder.Event(plugin, corev1.EventTypeNormal, "Apply", string(kc.Output))
		if err != nil || len(kc.Error) != 0 {
			e := string(kc.Error)

			c.recorder.Event(plugin, corev1.EventTypeWarning, "ApplyError", e)
			return fmt.Errorf("Error creating plugin: %s", e)
		}
	}

	return nil
}

// enqueuePlugin enqueues the plugin on add or delete
func (c *PluginController) enqueuePlugin(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Error(err)
		return
	}

	c.workqueue.AddRateLimited(key)
}
