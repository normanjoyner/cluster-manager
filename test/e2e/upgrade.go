package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"os"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/containership/cloud-agent/pkg/constants"
	"github.com/containership/cloud-agent/pkg/k8sutil"
	"github.com/containership/cloud-agent/pkg/log"

	provisioncsv3 "github.com/containership/cloud-agent/pkg/apis/provision.containership.io/v3"
)

var (
	cleanup bool
	loop    bool
)

func main() {
	flag.Parse()
	targetVersions := flag.Args()

	if len(targetVersions) == 0 {
		printUsage("At least one target version must be specified")
		os.Exit(1)
	}

	// We could pull in the kubeadm utils to check for valid versions, but that
	// seems like overkill. Since the leading 'v' has been a source of trouble,
	// at least make sure that's there.
	for _, version := range targetVersions {
		if version[0] != 'v' {
			printUsage("Version strings must start with 'v'")
			os.Exit(1)
		}
	}

	if err := run(targetVersions, loop); err != nil {
		log.Errorf("Test failed: %s", err)
		os.Exit(1)
	}

	log.Infof("Test passed!")
	os.Exit(0)
}

const (
	pollIntervalSeconds = 5

	// Actual timeout used in ClusterUpgrade. Upgrade retries 10x with 30s delay
	// between retries, so (30s + ~30s upgrade time) * 10 seems fair.
	nodeTimeoutSeconds = 600

	// Grace period after actual timeout to decide that this is a test failure
	// because a node did not get marked as finished.
	nodeTimeoutBufferSeconds = 60
)

func printUsage(details string) {
	// Don't use logger for usage because it looks weird and dumps stack trace
	fmt.Fprintf(os.Stderr, "Usage: go run upgrade.go <version-1> <version-2> ...\n")
	fmt.Fprintf(os.Stderr, "%s\n", details)
}

func init() {
	flag.BoolVar(&loop, "loop", true, "whether to loop over list of versions")
	flag.BoolVar(&cleanup, "cleanup", true, "clean up any created resources")
}

// TODO put this somewhere useful
func createClusterUpgrade(targetVersion string, id string) (*provisioncsv3.ClusterUpgrade, error) {
	log.Infof("Creating ClusterUpgrade %q with target version %s", id, targetVersion)

	labels := constants.BuildContainershipLabelMap(nil)

	cup, err := k8sutil.CSAPI().Client().ContainershipProvisionV3().ClusterUpgrades(constants.ContainershipNamespace).Create(&provisioncsv3.ClusterUpgrade{
		ObjectMeta: metav1.ObjectMeta{
			Name:   id,
			Labels: labels,
		},
		Spec: provisioncsv3.ClusterUpgradeSpec{
			ID:                 id,
			Type:               provisioncsv3.UpgradeTypeKubernetes,
			AddedAt:            "TODO",
			Description:        id,
			TargetVersion:      targetVersion,
			LabelSelector:      nil, // all nodes
			NodeTimeoutSeconds: nodeTimeoutSeconds,
		}})

	return cup, err
}

// TODO put this somewhere useful
func deleteClusterUpgrade(upgradeName string) error {
	log.Infof("Deleting ClusterUpgrade %q", upgradeName)
	return k8sutil.CSAPI().Client().ContainershipProvisionV3().ClusterUpgrades(constants.ContainershipNamespace).Delete(upgradeName, &metav1.DeleteOptions{})
}

// TODO polling is really stupid. We should use a watch / informer, but that
// wasn't working for some reason
func pollUpgrade(upgradeName string) provisioncsv3.UpgradeStatus {
	for {
		cup, err := k8sutil.CSAPI().Client().ContainershipProvisionV3().
			ClusterUpgrades(constants.ContainershipNamespace).Get(upgradeName, metav1.GetOptions{})
		if err != nil {
			// TODO there's some weird ephemeral permissions error that happens
			// and breaks things, just retry for now
			continue
		}

		// If the overall status is done then we're done - return that status
		log.Debugf("Cluster upgrade %q has cluster status %q", upgradeName, cup.Spec.Status.ClusterStatus)
		switch cup.Spec.Status.ClusterStatus {
		case provisioncsv3.UpgradeSuccess, provisioncsv3.UpgradeFailed:
			log.Infof("Cluster upgrade %q finished with cluster status %q", upgradeName, cup.Spec.Status.ClusterStatus)
			return cup.Spec.Status.ClusterStatus
		}

		// If an individual node failed then return that (fail fast)
		// Also check if any node is in progress for additional checking later
		nodeInProgress := ""
		for nodeName, nodeStatus := range cup.Spec.Status.NodeStatuses {
			if nodeStatus == provisioncsv3.UpgradeFailed {
				log.Errorf("Cluster upgrade %q failed for node %q", upgradeName, nodeName)
				return nodeStatus
			}

			if nodeStatus == provisioncsv3.UpgradeInProgress {
				log.Debugf("Cluster upgrade %q has InProgress node %q", upgradeName, nodeName)
				nodeInProgress = nodeName
			}
		}

		// Check for timeout - if a node times out, then its status should be
		// updated to Failed after nodeTimeoutSeconds, but we're testing that
		// functionality :)
		// Add some additional buffer to the real timeout to give it a chance
		// to fail properly
		// Note that this assumes that the start time is set properly in the status
		if nodeInProgress != "" {
			startTime, _ := time.Parse(time.UnixDate, cup.Spec.Status.CurrentStartTime)
			timeoutDuration := (nodeTimeoutSeconds + nodeTimeoutBufferSeconds) * time.Second
			if !startTime.IsZero() && time.Since(startTime) > timeoutDuration {
				log.Errorf("Upgrade %q failed to succeed or time out for node %q", upgradeName, nodeInProgress)
				return provisioncsv3.UpgradeFailed
			}
		}

		// Nothing interesting, keep chugging
		time.Sleep(pollIntervalSeconds * time.Second)
	}
}

func run(targetVersions []string, loop bool) error {
	sequenceNumber := 0
	for {
		for _, targetVersion := range targetVersions {
			uuid := generateRandomUUID()
			id := fmt.Sprintf("%s-%d-%s", targetVersion, sequenceNumber, uuid)
			_, err := createClusterUpgrade(targetVersion, id)
			if err != nil {
				return err
			}

			log.Infof("Polling upgrade %q until done", id)
			if result := pollUpgrade(id); result == provisioncsv3.UpgradeFailed {
				err = fmt.Errorf("Upgrade %q failed", id)
			}

			// Clean up if cleanup is enabled
			if cleanup {
				deleteClusterUpgrade(id)
			}

			if err != nil {
				return err
			}

			if !loop {
				return nil
			}

			sequenceNumber++
		}
	}
}

// generateRandomUUID generates a random UUID. It is actually just random; that
// is, it does not conform to any real spec.
func generateRandomUUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}

	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
