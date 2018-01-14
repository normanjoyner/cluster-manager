package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"

	containershipv3 "github.com/containership/cloud-agent/pkg/apis/containership.io/v3"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var emptyRegistrySpec = containershipv3.RegistrySpec{}
var emptyRegistry = &containershipv3.Registry{}

var registry1spec = containershipv3.RegistrySpec{
	ID:            "1",
	Description:   "description 1",
	Organization:  "1234-234-567",
	Email:         "",
	Serveraddress: "hub.docker.com",
	Provider:      "amazon_ec2_registry",
	Owner:         "",
}

var registry2spec = containershipv3.RegistrySpec{
	ID:            "2",
	Description:   "description 2",
	Organization:  "4321-657-4566",
	Email:         "asdf@gmail.com",
	Serveraddress: "ecr.aws.com",
	Provider:      "amazon_ec2_registry",
	Owner:         "",
}

var registry1 = &containershipv3.Registry{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "registry1",
		Namespace: "containership",
	},
	Spec: registry1spec,
}

var registry2 = &containershipv3.Registry{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "registry2",
		Namespace: "containership",
	},
	Spec: registry2spec,
}

func TestRegistryIsEqual(t *testing.T) {
	c := NewCsRegistries()
	// check for both being empty
	emptySameTest, err := c.IsEqual(emptyRegistrySpec, emptyRegistry)
	assert.Nil(t, err)
	assert.Equal(t, emptySameTest, true)
	//
	// check with spec being empty, and registry being empty
	emptyDiffTest, err := c.IsEqual(registry1spec, emptyRegistry)
	assert.Nil(t, err)
	assert.Equal(t, emptyDiffTest, false)
	emptyDiffTest2, err := c.IsEqual(emptyRegistrySpec, registry1)
	assert.Nil(t, err)
	assert.Equal(t, emptyDiffTest2, false)

	// check same with different keys same length
	differentTest, err := c.IsEqual(registry2spec, registry1)
	assert.Nil(t, err)
	assert.Equal(t, differentTest, false)

	// assert they are the same
	sameTest, err := c.IsEqual(registry2spec, registry2)
	assert.Nil(t, err)
	assert.Equal(t, sameTest, true)

	same, err := c.IsEqual(registry1spec, registry1)
	assert.Nil(t, err)
	assert.Equal(t, same, true)
}
