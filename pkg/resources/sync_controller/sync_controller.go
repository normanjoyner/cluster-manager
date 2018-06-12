package synccontroller

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	csclientset "github.com/containership/cloud-agent/pkg/client/clientset/versioned"

	"github.com/containership/cloud-agent/pkg/constants"
	"github.com/containership/cloud-agent/pkg/env"
	"github.com/containership/cloud-agent/pkg/log"
)

type syncController struct {
	name string

	// clientset is a clientset for our own API group
	clientset csclientset.Interface

	synced   cache.InformerSynced
	informer cache.SharedIndexInformer

	recorder record.EventRecorder
}

func (c *syncController) syncWithCloud(doSync func(), stopCh <-chan struct{}) error {
	log.Infof("Starting %s resource controller", c.name)
	log.Infof("Waiting for %s informer caches to sync", c.name)

	if ok := cache.WaitForCacheSync(stopCh, c.synced); !ok {
		return fmt.Errorf("Failed to wait for %s cache to sync", c.name)
	}

	// Only run one worker because a resource's underlying
	// cache is not thread-safe and we don't want to do parallel
	// requests to the API anyway
	go wait.JitterUntil(doSync,
		env.ContainershipCloudSyncInterval(),
		constants.SyncJitterFactor,
		true, // sliding: restart period only after doSync finishes
		stopCh)

	<-stopCh
	log.Infof("%s sync stopped", c.name)

	return nil
}
