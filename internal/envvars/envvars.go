package envvars

import (
	"os"
	"strconv"
	"strings"

	"github.com/containership/cloud-agent/internal/log"
	"time"
)

var agentSyncIntervalInSeconds int
var cloudClusterAPIKey string
var baseURL string
var clusterID string
var csHome string
var csCloudEnvironment string
var csServerPort string
var organizationID string
var kubeconfig string

func init() {
	organizationID = os.Getenv("CONTAINERSHIP_CLOUD_ORGANIZATION_ID")
	if organizationID == "" {
		log.Warn("CONTAINERSHIP_CLOUD_ORGANIZATION_ID env var not specified")
	}

	clusterID = os.Getenv("CONTAINERSHIP_CLOUD_CLUSTER_ID")
	if clusterID == "" {
		log.Warn("CONTAINERSHIP_CLOUD_CLUSTER_ID env var not specified")
	}

	cloudClusterAPIKey = os.Getenv("CONTAINERSHIP_CLOUD_CLUSTER_API_KEY")
	if cloudClusterAPIKey == "" {
		log.Warn("CONTAINERSHIP_CLOUD_CLUSTER_API_KEY env var not specified")
	}

	csHome = os.Getenv("CONTAINERSHIP_HOME")
	if csHome == "" {
		csHome = "/opt/containership/home"
		log.Info("CONTAINERSHIP_HOME env var not specified, defaulting to", csHome)
	}

	csCloudEnvironment = strings.ToLower(os.Getenv("CONTAINERSHIP_CLOUD_ENVIRONMENT"))
	if csCloudEnvironment == "" || csCloudEnvironment == "development" {
		baseURL = "https://stage-api.containership.io"
	} else {
		baseURL = "https://api.containership.io"
	}

	var err error
	agentSyncIntervalInSeconds, err = strconv.Atoi((os.Getenv("AGENT_SYNC_INTERVAL_IN_SECONDS")))
	if err != nil || agentSyncIntervalInSeconds == 0 {
		agentSyncIntervalInSeconds = 12
	}

	csServerPort = os.Getenv("CONTAINERSHIP_CLOUD_SERVER_PORT")
	if csServerPort == "" {
		csServerPort = "8000"
	}

	kubeconfig = os.Getenv("KUBECONFIG")
}

// GetOrganizationID returns Containership Cloud organization id
func GetOrganizationID() string {
	return organizationID
}

// GetClusterID returns Containership Cloud cluster id
func GetClusterID() string {
	return clusterID
}

// GetCloudClusterAPIKey returns Containership Cloud cluster api key
func GetCloudClusterAPIKey() string {
	return cloudClusterAPIKey
}

// GetBaseURL returns Containership Cloud API url
func GetBaseURL() string {
	return baseURL
}

// GetAgentSyncIntervalInSeconds returns agent sync interval in seconds
func GetAgentSyncIntervalInSeconds() time.Duration {
	return time.Duration(agentSyncIntervalInSeconds)
}

// GetCSHome returns Containership home directory
func GetCSHome() string {
	return csHome
}

// GetCSCloudEnvironment returns Containership Cloud environment
func GetCSCloudEnvironment() string {
	return csCloudEnvironment
}

// GetCSServerPort returns cloud-agent http server port
func GetCSServerPort() string {
	return csServerPort
}

// GetKubeconfig returns kubeconfig file if defined
func GetKubeconfig() string {
	return kubeconfig
}
