package env

import (
	"os"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"

	"github.com/containership/cloud-agent/internal/log"
)

type environment struct {
	csCloudSyncInterval             time.Duration
	agentInformerSyncInterval       time.Duration
	coordinatorInformerSyncInterval time.Duration
	cloudClusterAPIKey              string
	apiBaseURL                      string
	provisionBaseURL                string
	clusterID                       string
	csCloudEnvironment              string
	csServerPort                    string
	nodeName                        string
	organizationID                  string
	kubeconfig                      string
	kubectlPath                     string
}

const (
	defaultAgentInformerSyncInterval       = time.Minute
	defaultCoordinatorInformerSyncInterval = time.Minute
	defaultContainershipCloudSyncInterval  = time.Second * 30
)

var env environment

func init() {
	// Note that logs for missing cloud env are at Info level because the
	// agent side does not require them so there's nothing to warn about.
	env.organizationID = os.Getenv("CONTAINERSHIP_CLOUD_ORGANIZATION_ID")
	if env.organizationID == "" {
		log.Debug("CONTAINERSHIP_CLOUD_ORGANIZATION_ID env var not specified")
	}

	env.clusterID = os.Getenv("CONTAINERSHIP_CLOUD_CLUSTER_ID")
	if env.clusterID == "" {
		log.Debug("CONTAINERSHIP_CLOUD_CLUSTER_ID env var not specified")
	}

	env.cloudClusterAPIKey = os.Getenv("CONTAINERSHIP_CLOUD_CLUSTER_API_KEY")
	if env.cloudClusterAPIKey == "" {
		log.Debug("CONTAINERSHIP_CLOUD_CLUSTER_API_KEY env var not specified")
	}

	env.csCloudEnvironment = strings.ToLower(os.Getenv("CONTAINERSHIP_CLOUD_ENVIRONMENT"))
	if env.csCloudEnvironment == "" {
		env.csCloudEnvironment = "production"
	}

	env.apiBaseURL = os.Getenv("CONTAINERSHIP_CLOUD_API_BASE_URL")
	if env.apiBaseURL == "" {
		env.apiBaseURL = "https://api.containership.io"
	}

	env.provisionBaseURL = os.Getenv("CONTAINERSHIP_CLOUD_PROVISION_BASE_URL")
	if env.provisionBaseURL == "" {
		env.provisionBaseURL = "https://provision.containership.io"
	}

	env.csCloudSyncInterval = getDurationEnvOrDefault("CONTAINERSHIP_CLOUD_SYNC_INTERVAL_SEC",
		defaultContainershipCloudSyncInterval)

	env.agentInformerSyncInterval = getDurationEnvOrDefault("AGENT_INFORMER_SYNC_INTERVAL_SEC",
		defaultAgentInformerSyncInterval)

	env.coordinatorInformerSyncInterval = getDurationEnvOrDefault("COORDINATOR_INFORMER_SYNC_INTERVAL_SEC",
		defaultCoordinatorInformerSyncInterval)

	env.csServerPort = os.Getenv("CONTAINERSHIP_CLOUD_SERVER_PORT")
	if env.csServerPort == "" {
		env.csServerPort = "8000"
	}

	env.kubectlPath = os.Getenv("KUBECTL_PATH")
	if env.kubectlPath == "" {
		env.kubectlPath = "kubectl"
	}

	env.kubeconfig = os.Getenv("KUBECONFIG")
	env.nodeName = os.Getenv("NODE_NAME")
}

// OrganizationID returns Containership Cloud organization id
func OrganizationID() string {
	return env.organizationID
}

// ClusterID returns Containership Cloud cluster id
func ClusterID() string {
	return env.clusterID
}

// CloudClusterAPIKey returns Containership Cloud cluster api key
func CloudClusterAPIKey() string {
	return env.cloudClusterAPIKey
}

// APIBaseURL returns Containership Cloud API url
func APIBaseURL() string {
	return env.apiBaseURL
}

// ProvisionBaseURL returns Containership Cloud Provision url
func ProvisionBaseURL() string {
	return env.provisionBaseURL
}

// ContainershipCloudSyncInterval returns the cloud sync interval
func ContainershipCloudSyncInterval() time.Duration {
	return env.csCloudSyncInterval
}

// AgentInformerSyncInterval returns the agent informer sync interval
func AgentInformerSyncInterval() time.Duration {
	return env.agentInformerSyncInterval
}

// CoordinatorInformerSyncInterval returns the coordinator informer sync interval
func CoordinatorInformerSyncInterval() time.Duration {
	return env.coordinatorInformerSyncInterval
}

// CSCloudEnvironment returns Containership Cloud environment
func CSCloudEnvironment() string {
	return env.csCloudEnvironment
}

// CSServerPort returns cloud-agent http server port
func CSServerPort() string {
	return env.csServerPort
}

// Kubeconfig returns kubeconfig file if defined
func Kubeconfig() string {
	return env.kubeconfig
}

// KubectlPath returns the path to use the kubectl command
func KubectlPath() string {
	return env.kubectlPath
}

// NodeName returns the name of the node that is running the process
func NodeName() string {
	return env.nodeName
}

// Dump dumps the environment if we're in development mode
func Dump() {
	if env.csCloudEnvironment == "development" {
		dump := spew.Sdump(env)
		log.Debug(dump)
	}
}

func getDurationEnvOrDefault(key string, defaultVal time.Duration) time.Duration {
	val, err := time.ParseDuration(os.Getenv(key))
	if err != nil || val <= 0 {
		val = defaultVal
	}
	return val
}
