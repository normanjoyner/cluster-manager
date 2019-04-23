package coordinator

import (
	"encoding/json"
	"fmt"

	"github.com/containership/cluster-manager/pkg/log"
	"github.com/containership/cluster-manager/pkg/request"
)

// NodeCloudStatusMessage is the message posted to Cloud to update a node status
type NodeCloudStatusMessage struct {
	Status NodeCloudStatus `json:"status"`
}

// NodeCloudStatus is a node status that Cloud uses
type NodeCloudStatus struct {
	// Type is the status type
	Type string `json:"type"`
	// Percent is the progress percentage within this Type
	Percent float32 `json:"percent"`
}

const (
	// NodeCloudStatusBootstrapping is not used for upgrade and is listed here for reference
	NodeCloudStatusBootstrapping = "BOOTSTRAPPING"
	// NodeCloudStatusRunning should be posted when a node upgrade completes
	NodeCloudStatusRunning = "RUNNING"
)

// postNodeCloudStatusMessage posts the node cloud status to cloud. Note that this
// status is not (currently) intended to be strictly identical to the actual
// node status in the ClusterUpgrade CRD and is only used for essentially a
// boolean "is running or not" check on cloud side.
func postNodeCloudStatusMessage(nodeID string, status *NodeCloudStatusMessage) error {
	path := fmt.Sprintf("/organizations/{{.OrganizationID}}/clusters/{{.ClusterID}}/nodes/%s/status", nodeID)

	body, err := json.Marshal(status)
	if err != nil {
		return err
	}

	req, err := request.New(request.CloudServiceProvision, path, "PUT", []byte(body))
	if err != nil {
		return err
	}

	resp, err := req.MakeRequest()
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	return nil
}

// postNodeCloudStatusMessageWithRetry posts the node cloud status, retrying up to
// numRetries times.
// TODO consider using a library such as `pester` to handle actual HTTP retries
// here and elsewhere
func postNodeCloudStatusMessageWithRetry(nodeID string, status *NodeCloudStatusMessage, numRetries int) error {
	var err error
	for attempt := 1; attempt <= numRetries; attempt++ {
		log.Debugf("PUT node cloud status attempt %d", attempt)
		if err = postNodeCloudStatusMessage(nodeID, status); err == nil {
			// Request was successful so we're done
			break
		}
	}

	return err
}
