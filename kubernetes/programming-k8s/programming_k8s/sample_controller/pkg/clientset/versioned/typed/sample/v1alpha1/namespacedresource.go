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

package v1alpha1

import (
	"context"
	json "encoding/json"
	"fmt"
	"time"

	v1alpha1 "github.com/wjiec/programming_k8s/sample_controller/pkg/apis/sample/v1alpha1"
	samplev1alpha1 "github.com/wjiec/programming_k8s/sample_controller/pkg/applyconfiguration/sample/v1alpha1"
	scheme "github.com/wjiec/programming_k8s/sample_controller/pkg/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// NamespacedResourcesGetter has a method to return a NamespacedResourceInterface.
// A group's client should implement this interface.
type NamespacedResourcesGetter interface {
	NamespacedResources(namespace string) NamespacedResourceInterface
}

// NamespacedResourceInterface has methods to work with NamespacedResource resources.
type NamespacedResourceInterface interface {
	Create(ctx context.Context, namespacedResource *v1alpha1.NamespacedResource, opts v1.CreateOptions) (*v1alpha1.NamespacedResource, error)
	Update(ctx context.Context, namespacedResource *v1alpha1.NamespacedResource, opts v1.UpdateOptions) (*v1alpha1.NamespacedResource, error)
	UpdateStatus(ctx context.Context, namespacedResource *v1alpha1.NamespacedResource, opts v1.UpdateOptions) (*v1alpha1.NamespacedResource, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.NamespacedResource, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.NamespacedResourceList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.NamespacedResource, err error)
	Apply(ctx context.Context, namespacedResource *samplev1alpha1.NamespacedResourceApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.NamespacedResource, err error)
	ApplyStatus(ctx context.Context, namespacedResource *samplev1alpha1.NamespacedResourceApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.NamespacedResource, err error)
	NamespacedResourceExpansion
}

// namespacedResources implements NamespacedResourceInterface
type namespacedResources struct {
	client rest.Interface
	ns     string
}

// newNamespacedResources returns a NamespacedResources
func newNamespacedResources(c *SampleV1alpha1Client, namespace string) *namespacedResources {
	return &namespacedResources{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the namespacedResource, and returns the corresponding namespacedResource object, and an error if there is any.
func (c *namespacedResources) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.NamespacedResource, err error) {
	result = &v1alpha1.NamespacedResource{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("namespacedresources").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of NamespacedResources that match those selectors.
func (c *namespacedResources) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.NamespacedResourceList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.NamespacedResourceList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("namespacedresources").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested namespacedResources.
func (c *namespacedResources) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("namespacedresources").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a namespacedResource and creates it.  Returns the server's representation of the namespacedResource, and an error, if there is any.
func (c *namespacedResources) Create(ctx context.Context, namespacedResource *v1alpha1.NamespacedResource, opts v1.CreateOptions) (result *v1alpha1.NamespacedResource, err error) {
	result = &v1alpha1.NamespacedResource{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("namespacedresources").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(namespacedResource).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a namespacedResource and updates it. Returns the server's representation of the namespacedResource, and an error, if there is any.
func (c *namespacedResources) Update(ctx context.Context, namespacedResource *v1alpha1.NamespacedResource, opts v1.UpdateOptions) (result *v1alpha1.NamespacedResource, err error) {
	result = &v1alpha1.NamespacedResource{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("namespacedresources").
		Name(namespacedResource.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(namespacedResource).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *namespacedResources) UpdateStatus(ctx context.Context, namespacedResource *v1alpha1.NamespacedResource, opts v1.UpdateOptions) (result *v1alpha1.NamespacedResource, err error) {
	result = &v1alpha1.NamespacedResource{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("namespacedresources").
		Name(namespacedResource.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(namespacedResource).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the namespacedResource and deletes it. Returns an error if one occurs.
func (c *namespacedResources) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("namespacedresources").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *namespacedResources) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("namespacedresources").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched namespacedResource.
func (c *namespacedResources) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.NamespacedResource, err error) {
	result = &v1alpha1.NamespacedResource{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("namespacedresources").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

// Apply takes the given apply declarative configuration, applies it and returns the applied namespacedResource.
func (c *namespacedResources) Apply(ctx context.Context, namespacedResource *samplev1alpha1.NamespacedResourceApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.NamespacedResource, err error) {
	if namespacedResource == nil {
		return nil, fmt.Errorf("namespacedResource provided to Apply must not be nil")
	}
	patchOpts := opts.ToPatchOptions()
	data, err := json.Marshal(namespacedResource)
	if err != nil {
		return nil, err
	}
	name := namespacedResource.Name
	if name == nil {
		return nil, fmt.Errorf("namespacedResource.Name must be provided to Apply")
	}
	result = &v1alpha1.NamespacedResource{}
	err = c.client.Patch(types.ApplyPatchType).
		Namespace(c.ns).
		Resource("namespacedresources").
		Name(*name).
		VersionedParams(&patchOpts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

// ApplyStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating ApplyStatus().
func (c *namespacedResources) ApplyStatus(ctx context.Context, namespacedResource *samplev1alpha1.NamespacedResourceApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.NamespacedResource, err error) {
	if namespacedResource == nil {
		return nil, fmt.Errorf("namespacedResource provided to Apply must not be nil")
	}
	patchOpts := opts.ToPatchOptions()
	data, err := json.Marshal(namespacedResource)
	if err != nil {
		return nil, err
	}

	name := namespacedResource.Name
	if name == nil {
		return nil, fmt.Errorf("namespacedResource.Name must be provided to Apply")
	}

	result = &v1alpha1.NamespacedResource{}
	err = c.client.Patch(types.ApplyPatchType).
		Namespace(c.ns).
		Resource("namespacedresources").
		Name(*name).
		SubResource("status").
		VersionedParams(&patchOpts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}