package envvars

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"

	"github.com/containership/cloud-agent/internal/log"
)

type environment struct {
	agentSyncIntervalInSeconds int
	cloudClusterAPIKey         string
	baseURL                    string
	clusterID                  string
	csCloudEnvironment         string
	csServerPort               string
	organizationID             string
	kubeconfig                 string
}

var env environment

func init() {
	// Note that logs for missing cloud env are at Info level because the
	// agent side does not require them so there's nothing to warn about.
	env.organizationID = os.Getenv("CONTAINERSHIP_CLOUD_ORGANIZATION_ID")
	if env.organizationID == "" {
		log.Info("CONTAINERSHIP_CLOUD_ORGANIZATION_ID env var not specified")
	}

	env.clusterID = os.Getenv("CONTAINERSHIP_CLOUD_CLUSTER_ID")
	if env.clusterID == "" {
		log.Info("CONTAINERSHIP_CLOUD_CLUSTER_ID env var not specified")
	}

	env.cloudClusterAPIKey = os.Getenv("CONTAINERSHIP_CLOUD_CLUSTER_API_KEY")
	if env.cloudClusterAPIKey == "" {
		log.Info("CONTAINERSHIP_CLOUD_CLUSTER_API_KEY env var not specified")
	}

	env.csCloudEnvironment = strings.ToLower(os.Getenv("CONTAINERSHIP_CLOUD_ENVIRONMENT"))
	if env.csCloudEnvironment == "development" {
		env.baseURL = "https://stage-api.containership.io"
	} else {
		// Default to production
		env.baseURL = "https://api.containership.io"
	}

	var err error
	env.agentSyncIntervalInSeconds, err = strconv.Atoi((os.Getenv("AGENT_SYNC_INTERVAL_IN_SECONDS")))
	if err != nil || env.agentSyncIntervalInSeconds == 0 {
		env.agentSyncIntervalInSeconds = 12
	}

	env.csServerPort = os.Getenv("CONTAINERSHIP_CLOUD_SERVER_PORT")
	if env.csServerPort == "" {
		env.csServerPort = "8000"
	}

	env.kubeconfig = os.Getenv("KUBECONFIG")
}

// GetOrganizationID returns Containership Cloud organization id
func GetOrganizationID() string {
	return env.organizationID
}

// GetClusterID returns Containership Cloud cluster id
func GetClusterID() string {
	return env.clusterID
}

// GetCloudClusterAPIKey returns Containership Cloud cluster api key
func GetCloudClusterAPIKey() string {
	return env.cloudClusterAPIKey
}

// GetBaseURL returns Containership Cloud API url
func GetBaseURL() string {
	return env.baseURL
}

// GetAgentSyncIntervalInSeconds returns agent sync interval in seconds
func GetAgentSyncIntervalInSeconds() time.Duration {
	return time.Duration(env.agentSyncIntervalInSeconds)
}

// GetCSCloudEnvironment returns Containership Cloud environment
func GetCSCloudEnvironment() string {
	return env.csCloudEnvironment
}

// GetCSServerPort returns cloud-agent http server port
func GetCSServerPort() string {
	return env.csServerPort
}

// GetKubeconfig returns kubeconfig file if defined
func GetKubeconfig() string {
	return env.kubeconfig
}

// DumpDevelopmentEnvironment dumps the environment if we're in development mode
func DumpDevelopmentEnvironment() {
	if env.csCloudEnvironment == "development" {
		dump := spew.Sdump(env)
		log.Debug(dump)
	}
}
