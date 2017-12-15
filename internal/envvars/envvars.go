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

func GetOrganizationID() string {
	return organizationID
}

func GetClusterID() string {
	return clusterID
}

func GetCloudClusterAPIKey() string {
	return cloudClusterAPIKey
}

func GetBaseURL() string {
	return baseURL
}

func GetAgentSyncIntervalInSeconds() int {
	return agentSyncIntervalInSeconds
}

func GetCSCloudEnvironment() string {
	return csCloudEnvironment
}

func GetCSServerPort() string {
	return csServerPort
}
