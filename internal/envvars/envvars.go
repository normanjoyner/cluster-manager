package envvars

import (
	"log"
	"os"
	"strconv"
)

var agentSyncIntervalInSeconds int
var cloudClusterAPIKey string
var baseURL string
var clusterID string
var csCloudEnvironment string
var csServerPort string
var organizationID string

func init() {
	organizationID = os.Getenv("CONTAINERSHIP_CLOUD_ORGANIZATION_ID")
	if organizationID == "" {
		log.Println("CONTAINERSHIP_CLOUD_ORGANIZATION_ID env var not specified")
	}

	clusterID = os.Getenv("CONTAINERSHIP_CLOUD_CLUSTER_ID")
	if clusterID == "" {
		log.Println("CONTAINERSHIP_CLOUD_CLUSTER_ID env var not specified")
	}

	cloudClusterAPIKey = os.Getenv("CONTAINERSHIP_CLOUD_CLUSTER_API_KEY")
	if cloudClusterAPIKey == "" {
		log.Println("CONTAINERSHIP_CLOUD_CLUSTER_API_KEY env var not specified")
	}

	csCloudEnvironment = os.Getenv("CONTAINERSHIP_CLOUD_BASE_URL")
	if csCloudEnvironment == "" || csCloudEnvironment == "Development" {
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
func GetAgentSyncIntervalInSeconds() int {
	return agentSyncIntervalInSeconds
}

// GetCSCloudEnvironment returns Containership Cloud environment
func GetCSCloudEnvironment() string {
	return csCloudEnvironment
}

// GetCSServerPort returns cloud-agent http server port
func GetCSServerPort() string {
	return csServerPort
}
