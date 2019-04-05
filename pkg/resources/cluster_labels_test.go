package resources

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	csv3 "github.com/containership/cluster-manager/pkg/apis/containership.io/v3"
	"github.com/containership/cluster-manager/pkg/constants"
)

var emptyClusterLabel = &csv3.ClusterLabel{}

var clusterLabel1 = &csv3.ClusterLabel{
	ObjectMeta: metav1.ObjectMeta{
		Name: "name0",
	},
	Spec: csv3.ClusterLabelSpec{
		ID:        "1",
		CreatedAt: "123",
		UpdatedAt: "456",
		Key:       constants.ContainershipClusterLabelPrefix + "key1",
		Value:     "value1",
	},
}

func TestClusterLabelIsEqual(t *testing.T) {
	c := NewCsClusterLabels(nil)

	eq, err := c.IsEqual(csv3.ClusterLabelSpec{}, emptyClusterLabel)
	assert.Nil(t, err, "both empty")
	assert.True(t, eq, "both empty")

	same := clusterLabel1.DeepCopy().Spec
	eq, err = c.IsEqual(same, clusterLabel1)
	assert.True(t, eq, "copied spec")

	diff := clusterLabel1.DeepCopy().Spec
	diff.ID = "different"
	eq, err = c.IsEqual(diff, clusterLabel1)
	assert.False(t, eq, "different ID")

	diff = clusterLabel1.DeepCopy().Spec
	diff.CreatedAt = "different"
	eq, err = c.IsEqual(diff, clusterLabel1)
	assert.False(t, eq, "different created_at")

	diff = clusterLabel1.DeepCopy().Spec
	diff.UpdatedAt = "different"
	eq, err = c.IsEqual(diff, clusterLabel1)
	assert.False(t, eq, "different updated_at")

	diff = clusterLabel1.DeepCopy().Spec
	diff.Key = "different"
	eq, err = c.IsEqual(diff, clusterLabel1)
	assert.False(t, eq, "different key")

	diff = clusterLabel1.DeepCopy().Spec
	diff.Value = "different"
	eq, err = c.IsEqual(diff, clusterLabel1)
	assert.False(t, eq, "different value")
}

func TestClusterLabelsCache(t *testing.T) {
	clusterLabelBytes := []byte(`[{
	"id" : "1234",
	"created_at" : "123",
	"updated_at" : "456",
	"key" : "key1",
	"value" : "value1"
}]`)

	u := NewCsClusterLabels(nil)

	err := json.Unmarshal(clusterLabelBytes, &u.cache)
	assert.NoError(t, err, "unmarshal good data")

	c := u.Cache()
	assert.Equal(t, u.cache, c)
}
