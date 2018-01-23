/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package fake

import (
	v3 "github.com/containership/cloud-agent/pkg/apis/containership.io/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakePlugins implements PluginInterface
type FakePlugins struct {
	Fake *FakeContainershipV3
	ns   string
}

var pluginsResource = schema.GroupVersionResource{Group: "containership.io", Version: "v3", Resource: "plugins"}

var pluginsKind = schema.GroupVersionKind{Group: "containership.io", Version: "v3", Kind: "Plugin"}

// Get takes name of the plugin, and returns the corresponding plugin object, and an error if there is any.
func (c *FakePlugins) Get(name string, options v1.GetOptions) (result *v3.Plugin, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(pluginsResource, c.ns, name), &v3.Plugin{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v3.Plugin), err
}

// List takes label and field selectors, and returns the list of Plugins that match those selectors.
func (c *FakePlugins) List(opts v1.ListOptions) (result *v3.PluginList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(pluginsResource, pluginsKind, c.ns, opts), &v3.PluginList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v3.PluginList{}
	for _, item := range obj.(*v3.PluginList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested plugins.
func (c *FakePlugins) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(pluginsResource, c.ns, opts))

}

// Create takes the representation of a plugin and creates it.  Returns the server's representation of the plugin, and an error, if there is any.
func (c *FakePlugins) Create(plugin *v3.Plugin) (result *v3.Plugin, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(pluginsResource, c.ns, plugin), &v3.Plugin{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v3.Plugin), err
}

// Update takes the representation of a plugin and updates it. Returns the server's representation of the plugin, and an error, if there is any.
func (c *FakePlugins) Update(plugin *v3.Plugin) (result *v3.Plugin, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(pluginsResource, c.ns, plugin), &v3.Plugin{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v3.Plugin), err
}

// Delete takes name of the plugin and deletes it. Returns an error if one occurs.
func (c *FakePlugins) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(pluginsResource, c.ns, name), &v3.Plugin{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakePlugins) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(pluginsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v3.PluginList{})
	return err
}

// Patch applies the patch and returns the patched plugin.
func (c *FakePlugins) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v3.Plugin, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(pluginsResource, c.ns, name, data, subresources...), &v3.Plugin{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v3.Plugin), err
}