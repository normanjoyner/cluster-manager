package coordinator

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/afero"

	"github.com/containership/cluster-manager/pkg/constants"
	"github.com/containership/cluster-manager/pkg/env"
	"github.com/containership/cluster-manager/pkg/k8sutil/kubectl"
	"github.com/containership/cluster-manager/pkg/log"
	"github.com/containership/cluster-manager/pkg/request"
	"github.com/containership/cluster-manager/pkg/tools"
	"github.com/containership/cluster-manager/pkg/tools/fsutil"

	csv3 "github.com/containership/cluster-manager/pkg/apis/containership.io/v3"
	csclientset "github.com/containership/cluster-manager/pkg/client/clientset/versioned"
	csinformers "github.com/containership/cluster-manager/pkg/client/informers/externalversions"
	cslisters "github.com/containership/cluster-manager/pkg/client/listers/containership.io/v3"

	kubeerror "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	watch "k8s.io/apimachinery/pkg/watch"
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
	kubeclientset kubernetes.Interface
	// clientset is a clientset for containership API group
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
		kubeclientset: kubeclientset,
		clientset:     clientset,
		workqueue:     workqueue.NewNamedRateLimitingQueue(rateLimiter, "Plugin"),
		recorder:      tools.CreateAndStartRecorder(kubeclientset, pluginControllerName),
	}

	// Instantiate resource informers
	pluginInformer := csInformerFactory.Containership().V3().Plugins()

	// namespace informer listens for add events an queues the namespace
	pluginInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: pc.enqueuePlugin,
		UpdateFunc: func(old, new interface{}) {
			newPlugin := new.(*csv3.Plugin)
			oldPlugin := old.(*csv3.Plugin)
			if newPlugin.ResourceVersion == oldPlugin.ResourceVersion {
				// Periodic resync will send update events for all known Plugins.
				// Two different versions of the same Plugin will always have different RVs.
				return
			}

			pc.enqueuePlugin(newPlugin)
		},
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
	return errors.Wrap(err, "syncing plugin failed")
}

// pluginSyncHandler uses kubectl to apply the plugin manifest if a plugin CRD
// exists, if the plugin CRD does not exist delete the associated plugin
// manifest files
func (c *PluginController) pluginSyncHandler(key string) error {
	_, name, _ := cache.SplitMetaNamespaceKey(key)
	plugin, err := c.pluginLister.Plugins(constants.ContainershipNamespace).Get(name)

	if err != nil {
		// If the plugin CRD does not exist delete the plugin manifest files
		if kubeerror.IsNotFound(err) {
			return c.deletePlugin(name)
		}

		return errors.Wrap(err, "getting plugin failed with error other than not found")
	}

	if plugin.Spec.Type == constants.ClusterManagementPluginType && env.IsClusterManagementPluginSyncDisabled() {
		return nil
	}

	log.Debugf("%s syncing plugin of type %q with implementation %q", pluginControllerName, plugin.Spec.Type, plugin.Spec.Implementation)
	// Create the plugin manifest files
	return c.applyPlugin(plugin)
}

func getPluginEndpoint(plugin *csv3.Plugin) string {
	previousVersion := getPreviousVersion(plugin)
	return fmt.Sprintf("/organizations/{{.OrganizationID}}/clusters/{{.ClusterID}}/plugins/%s?previous_version=%s", plugin.Spec.ID, previousVersion)
}

func getPreviousVersion(plugin *csv3.Plugin) string {
	annotation, ok := plugin.Annotations[constants.PluginHistoryAnnotation]
	if !ok {
		return ""
	}

	history := make([]csv3.PluginSpec, 0)
	err := json.Unmarshal([]byte(annotation), &history)

	if err != nil || len(history) == 0 {
		return ""
	}

	lastRevision := history[len(history)-1]
	return lastRevision.Version
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
func deletePluginUsingFiles(pluginID, pluginPath string, fileManager pluginFileManager) error {
	files, err := ioutil.ReadDir(pluginPath)
	if err != nil {
		return errors.Wrap(err, "unable to read plugin path directory")
	}

	for _, f := range files {
		fullFilename := fileManager.pluginManifestsFilePath(f.Name())

		var kc *kubectl.Kubectl
		kc = kubectl.NewDeleteCmd(fullFilename)

		err = runAndLogFromKubectl(kc)
		if err != nil {
			return errors.Wrap(err, "deleting plugin using files failed")
		}
	}

	return nil
}

func (c *PluginController) deletePlugin(name string) error {
	fileManager := pluginFileManager{
		name,
	}
	pluginPath := fileManager.pluginManifestsDir()

	var kc *kubectl.Kubectl
	if stat, err := os.Stat(pluginPath); os.IsNotExist(err) || !stat.IsDir() {
		// Covers edge case that if plugin does not have a directory with manifests
		// it will try and delete by the containership.io/plugin-id label
		label := pluginLabelKey + "=" + name
		kc = kubectl.NewDeleteByLabelCmd(constants.ContainershipNamespace, label)

		err := runAndLogFromKubectl(kc)
		if err != nil {
			return errors.Wrap(err, "deleting plugin using labels failed")
		}

		return nil
	}

	err := deletePluginUsingFiles(name, pluginPath, fileManager)
	if err != nil {
		return errors.Wrap(err, "delete plugin failed")
	}

	// attempts to clean up the manifest directory that was created,
	// if there is an error it's ignored
	fileManager.cleanUpPluginManifests()
	return nil
}

type jsonPluginsResponse struct {
	Manifests [][]interface{} `json:"manifests"`
	Jobs      jobs            `json:"jobs,omitempty"`
}

type jobs struct {
	PreApply  []*batchv1.Job `json:"pre_apply,omitempty"`
	PostApply []*batchv1.Job `json:"post_apply,omitempty"`
}

// getPlugin gets the plugin spec from cloud, or returns an error
func (c *PluginController) getPlugin(p *csv3.Plugin) (*jsonPluginsResponse, error) {
	path := getPluginEndpoint(p)
	bytes, err := makeRequest(path)
	if err != nil {
		return nil, err
	}

	var plugin jsonPluginsResponse
	plugin.Jobs.PreApply = make([]*batchv1.Job, 0)
	plugin.Jobs.PostApply = make([]*batchv1.Job, 0)

	err = json.Unmarshal(bytes, &plugin)
	if err != nil {
		return nil, err
	}

	return &plugin, nil
}

// formatPlugin takes the manifests from the plugin spec returned from cloud
// and splits them up to be written to files under the plugins/:plugin_id. It
// then returns an array of strings, which are the file paths.
func formatPlugin(spec csv3.PluginSpec, pluginDetails *jsonPluginsResponse, fileManager pluginFileManager) ([]string, error) {
	manifests := make([]string, 0)

	for index, resources := range pluginDetails.Manifests {
		manifestBundle := make([]byte, 0)
		for _, r := range resources {
			// Re-Marshal the generic object so we can have each
			// as []byte
			unstructuredBytes, _ := json.Marshal(r)
			manifestBundle = append(manifestBundle, unstructuredBytes...)
		}

		filename := fileManager.pluginManifestsFilePath(buildPluginFilename(index, "json"))

		err := writePluginManifests(filename, manifestBundle)
		// we only want to short circuit and return if there was an error
		// else continue processing the manifests
		if err != nil {
			return nil, err
		}

		manifests = append(manifests, filename)
	}

	return manifests, nil
}

func pluginBaseDir() string {
	return "/plugins"
}

// pluginFileManager uses the plugins id to create and mantain the plugin
// directory for applying and updating plugins
type pluginFileManager struct {
	ID string
}

func (p pluginFileManager) initializePluginDirectories() error {
	md := p.pluginManifestsDir()
	err := osFs.MkdirAll(md, os.ModePerm)
	if err != nil {
		return errors.Wrap(err, "initializing plugin directories failed")
	}

	return nil
}

func (p pluginFileManager) pluginDir() string {
	return path.Join(pluginBaseDir(), p.ID)
}

func (p pluginFileManager) pluginManifestsDir() string {
	return path.Join(p.pluginDir(), "manifests")
}

func (p pluginFileManager) pluginManifestsFilePath(name string) string {
	return path.Join(p.pluginManifestsDir(), name)
}

func buildPluginFilename(name int, fileType string) string {
	return fmt.Sprintf("%d.%s", name, fileType)
}

// writePluginManifests takes the manifests and writes them to a .json file.
// They are named as their index so that if `kubectl apply -f` is run on
// the directory they will be applied in the correct order, since kubectl apply
// runs files in alphanumeric order
func writePluginManifests(filename string, manifest []byte) error {
	return ioutil.WriteFile(filename, manifest, 0644)
}

// cleanUpPluginManifests is a best effort function to clean up a plugins
// directory by removing all manifests written for the plugin.
// we log the error but don't return it, or retry
func (p pluginFileManager) cleanUpPluginManifests() {
	err := osFs.RemoveAll(p.pluginDir())
	if err != nil {
		log.Error(err)
	}
}

// applyPlugin takes the manifests under the plugins id and runs them through
// kubectl apply
func (c *PluginController) applyPlugin(plugin *csv3.Plugin) error {
	pluginDetails, err := c.getPlugin(plugin)
	if err != nil {
		return errors.Wrap(err, "get request to Containership api failed")
	}

	fileManager := pluginFileManager{
		plugin.Spec.ID,
	}
	fileManager.cleanUpPluginManifests()
	fileManager.initializePluginDirectories()

	if pluginDetails.Manifests != nil {
		_, err := formatPlugin(plugin.Spec, pluginDetails, fileManager)

		if err != nil {
			return errors.Wrap(err, "formatting manifests for plugin failed")
		}
	}

	// apply jobs for applying before manifests
	err = c.runJob(pluginDetails.Jobs.PreApply, plugin)
	if err != nil {
		return errors.Wrap(err, "running pre apply jobs failed")
	}

	err = c.applyManifests(fileManager, plugin)
	if err != nil {
		return errors.Wrap(err, "applying manifests failed")
	}

	// apply jobs for clean up after manifests
	err = c.runJob(pluginDetails.Jobs.PostApply, plugin)
	if err != nil {
		return errors.Wrap(err, "running post apply jobs failed")
	}

	return nil
}

func (c *PluginController) applyManifests(fileManager pluginFileManager, plugin *csv3.Plugin) error {
	manifests := fileManager.pluginManifestsDir()
	if empty, err := fsutil.IsEmpty(manifests); err != nil {
		return errors.Wrap(err, "checking plugin directory for is empty failed")
	} else if empty {
		return nil
	}

	kc := kubectl.NewApplyCmd(manifests)

	err := kc.Run()

	c.recorder.Event(plugin, corev1.EventTypeNormal, "Apply", string(kc.Output))

	if err != nil {
		errMsg := fmt.Sprintf("Error creating plugin: %s", err)
		if e := string(kc.Error); e != "" {
			errMsg = fmt.Sprintf("%s, kubectl stderr: %s", errMsg, e)
		}

		c.recorder.Event(plugin, corev1.EventTypeWarning, "ApplyError", errMsg)
		return fmt.Errorf("Error creating plugin: %s", errMsg)
	}

	return nil
}

func (c *PluginController) runJob(jobs []*batchv1.Job, plugin *csv3.Plugin) error {
	if len(jobs) <= 0 {
		return nil
	}

	var jobErr error
	for _, job := range jobs {
		if job.Spec.ActiveDeadlineSeconds == nil {
			return fmt.Errorf("ActiveDeadlineSeconds needs to be specified for any plugin job")
		}

		if job.Spec.BackoffLimit == nil {
			limit := int32(1)
			job.Spec.BackoffLimit = &limit
		}

		_, err := c.kubeclientset.BatchV1().Jobs(constants.ContainershipNamespace).Create(job)
		if err != nil {
			return errors.Wrap(err, "creating job failed")
		}

		w, err := c.kubeclientset.BatchV1().Jobs(constants.ContainershipNamespace).Watch(metav1.ListOptions{
			FieldSelector: fields.OneTermEqualSelector("metadata.name", job.Name).String(),
		})
		if err != nil {
			return errors.Wrap(err, "getting watcher for job failed")
		}

		ch := w.ResultChan()
		for event := range ch {
			j, ok := event.Object.(*batchv1.Job)
			if !ok {
				continue
			}

			switch event.Type {
			case watch.Modified:
				if len(j.Status.Conditions) > 0 {
					failed, reason, _ := jobDidFail(j)
					// If a job is not able to finish from exceeding BackoffLimit or ActiveDeadlineSeconds
					// it will change its status to failed
					if failed {
						jobErr = fmt.Errorf("Job finished with failed status %s", reason)
					}

					w.Stop()
				}
			case watch.Deleted:
				// if job is deleted we should stop watching
				w.Stop()
			default:
				log.Debugf("Plugin job '%s' received event of type %q", job.Name, event.Type)
			}
		}

		// deleting is best effort, if it fails that's fine, also
		// the process shouldn't be blocking
		propagationPolicy := metav1.DeletePropagationForeground
		go c.kubeclientset.BatchV1().Jobs(constants.ContainershipNamespace).Delete(job.Name, &metav1.DeleteOptions{
			PropagationPolicy: &propagationPolicy,
		})
	}

	return jobErr
}

func jobDidFail(job *batchv1.Job) (bool, string, error) {
	if len(job.Status.Conditions) == 0 {
		return false, "", fmt.Errorf("Job must have at least one status condition to check to determine failure")
	}

	for _, condition := range job.Status.Conditions {
		if condition.Type == batchv1.JobFailed {
			return true, condition.Reason, nil
		}
	}

	return false, "", nil
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
