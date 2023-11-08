/*
Copyright (c) 2023 Jayson Wang

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"
	json "encoding/json"
	"fmt"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"

	v1alpha2 "github.com/wjiec/programming_k8s/sample-controller/pkg/apis/sample/v1alpha2"
	samplev1alpha2 "github.com/wjiec/programming_k8s/sample-controller/pkg/applyconfiguration/sample/v1alpha2"
)

// FakeNamespacedResources implements NamespacedResourceInterface
type FakeNamespacedResources struct {
	Fake *FakeSampleV1alpha2
	ns   string
}

var namespacedresourcesResource = v1alpha2.SchemeGroupVersion.WithResource("namespacedresources")

var namespacedresourcesKind = v1alpha2.SchemeGroupVersion.WithKind("NamespacedResource")

// Get takes name of the namespacedResource, and returns the corresponding namespacedResource object, and an error if there is any.
func (c *FakeNamespacedResources) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha2.NamespacedResource, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(namespacedresourcesResource, c.ns, name), &v1alpha2.NamespacedResource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha2.NamespacedResource), err
}

// List takes label and field selectors, and returns the list of NamespacedResources that match those selectors.
func (c *FakeNamespacedResources) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha2.NamespacedResourceList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(namespacedresourcesResource, namespacedresourcesKind, c.ns, opts), &v1alpha2.NamespacedResourceList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha2.NamespacedResourceList{ListMeta: obj.(*v1alpha2.NamespacedResourceList).ListMeta}
	for _, item := range obj.(*v1alpha2.NamespacedResourceList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested namespacedResources.
func (c *FakeNamespacedResources) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(namespacedresourcesResource, c.ns, opts))

}

// Create takes the representation of a namespacedResource and creates it.  Returns the server's representation of the namespacedResource, and an error, if there is any.
func (c *FakeNamespacedResources) Create(ctx context.Context, namespacedResource *v1alpha2.NamespacedResource, opts v1.CreateOptions) (result *v1alpha2.NamespacedResource, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(namespacedresourcesResource, c.ns, namespacedResource), &v1alpha2.NamespacedResource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha2.NamespacedResource), err
}

// Update takes the representation of a namespacedResource and updates it. Returns the server's representation of the namespacedResource, and an error, if there is any.
func (c *FakeNamespacedResources) Update(ctx context.Context, namespacedResource *v1alpha2.NamespacedResource, opts v1.UpdateOptions) (result *v1alpha2.NamespacedResource, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(namespacedresourcesResource, c.ns, namespacedResource), &v1alpha2.NamespacedResource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha2.NamespacedResource), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeNamespacedResources) UpdateStatus(ctx context.Context, namespacedResource *v1alpha2.NamespacedResource, opts v1.UpdateOptions) (*v1alpha2.NamespacedResource, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(namespacedresourcesResource, "status", c.ns, namespacedResource), &v1alpha2.NamespacedResource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha2.NamespacedResource), err
}

// Delete takes name of the namespacedResource and deletes it. Returns an error if one occurs.
func (c *FakeNamespacedResources) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(namespacedresourcesResource, c.ns, name, opts), &v1alpha2.NamespacedResource{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeNamespacedResources) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(namespacedresourcesResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha2.NamespacedResourceList{})
	return err
}

// Patch applies the patch and returns the patched namespacedResource.
func (c *FakeNamespacedResources) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha2.NamespacedResource, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(namespacedresourcesResource, c.ns, name, pt, data, subresources...), &v1alpha2.NamespacedResource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha2.NamespacedResource), err
}

// Apply takes the given apply declarative configuration, applies it and returns the applied namespacedResource.
func (c *FakeNamespacedResources) Apply(ctx context.Context, namespacedResource *samplev1alpha2.NamespacedResourceApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha2.NamespacedResource, err error) {
	if namespacedResource == nil {
		return nil, fmt.Errorf("namespacedResource provided to Apply must not be nil")
	}
	data, err := json.Marshal(namespacedResource)
	if err != nil {
		return nil, err
	}
	name := namespacedResource.Name
	if name == nil {
		return nil, fmt.Errorf("namespacedResource.Name must be provided to Apply")
	}
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(namespacedresourcesResource, c.ns, *name, types.ApplyPatchType, data), &v1alpha2.NamespacedResource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha2.NamespacedResource), err
}

// ApplyStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating ApplyStatus().
func (c *FakeNamespacedResources) ApplyStatus(ctx context.Context, namespacedResource *samplev1alpha2.NamespacedResourceApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha2.NamespacedResource, err error) {
	if namespacedResource == nil {
		return nil, fmt.Errorf("namespacedResource provided to Apply must not be nil")
	}
	data, err := json.Marshal(namespacedResource)
	if err != nil {
		return nil, err
	}
	name := namespacedResource.Name
	if name == nil {
		return nil, fmt.Errorf("namespacedResource.Name must be provided to Apply")
	}
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(namespacedresourcesResource, c.ns, *name, types.ApplyPatchType, data, "status"), &v1alpha2.NamespacedResource{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha2.NamespacedResource), err
}