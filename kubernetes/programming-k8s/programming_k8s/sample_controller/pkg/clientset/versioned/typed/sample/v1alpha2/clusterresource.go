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

package v1alpha2

import (
	"context"
	json "encoding/json"
	"fmt"
	"time"

	v1alpha2 "github.com/wjiec/programming_k8s/sample_controller/pkg/apis/sample/v1alpha2"
	samplev1alpha2 "github.com/wjiec/programming_k8s/sample_controller/pkg/applyconfiguration/sample/v1alpha2"
	scheme "github.com/wjiec/programming_k8s/sample_controller/pkg/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// ClusterResourcesGetter has a method to return a ClusterResourceInterface.
// A group's client should implement this interface.
type ClusterResourcesGetter interface {
	ClusterResources() ClusterResourceInterface
}

// ClusterResourceInterface has methods to work with ClusterResource resources.
type ClusterResourceInterface interface {
	Create(ctx context.Context, clusterResource *v1alpha2.ClusterResource, opts v1.CreateOptions) (*v1alpha2.ClusterResource, error)
	Update(ctx context.Context, clusterResource *v1alpha2.ClusterResource, opts v1.UpdateOptions) (*v1alpha2.ClusterResource, error)
	UpdateStatus(ctx context.Context, clusterResource *v1alpha2.ClusterResource, opts v1.UpdateOptions) (*v1alpha2.ClusterResource, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha2.ClusterResource, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha2.ClusterResourceList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha2.ClusterResource, err error)
	Apply(ctx context.Context, clusterResource *samplev1alpha2.ClusterResourceApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha2.ClusterResource, err error)
	ApplyStatus(ctx context.Context, clusterResource *samplev1alpha2.ClusterResourceApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha2.ClusterResource, err error)
	ClusterResourceExpansion
}

// clusterResources implements ClusterResourceInterface
type clusterResources struct {
	client rest.Interface
}

// newClusterResources returns a ClusterResources
func newClusterResources(c *SampleV1alpha2Client) *clusterResources {
	return &clusterResources{
		client: c.RESTClient(),
	}
}

// Get takes name of the clusterResource, and returns the corresponding clusterResource object, and an error if there is any.
func (c *clusterResources) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha2.ClusterResource, err error) {
	result = &v1alpha2.ClusterResource{}
	err = c.client.Get().
		Resource("clusterresources").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of ClusterResources that match those selectors.
func (c *clusterResources) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha2.ClusterResourceList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha2.ClusterResourceList{}
	err = c.client.Get().
		Resource("clusterresources").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested clusterResources.
func (c *clusterResources) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("clusterresources").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a clusterResource and creates it.  Returns the server's representation of the clusterResource, and an error, if there is any.
func (c *clusterResources) Create(ctx context.Context, clusterResource *v1alpha2.ClusterResource, opts v1.CreateOptions) (result *v1alpha2.ClusterResource, err error) {
	result = &v1alpha2.ClusterResource{}
	err = c.client.Post().
		Resource("clusterresources").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(clusterResource).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a clusterResource and updates it. Returns the server's representation of the clusterResource, and an error, if there is any.
func (c *clusterResources) Update(ctx context.Context, clusterResource *v1alpha2.ClusterResource, opts v1.UpdateOptions) (result *v1alpha2.ClusterResource, err error) {
	result = &v1alpha2.ClusterResource{}
	err = c.client.Put().
		Resource("clusterresources").
		Name(clusterResource.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(clusterResource).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *clusterResources) UpdateStatus(ctx context.Context, clusterResource *v1alpha2.ClusterResource, opts v1.UpdateOptions) (result *v1alpha2.ClusterResource, err error) {
	result = &v1alpha2.ClusterResource{}
	err = c.client.Put().
		Resource("clusterresources").
		Name(clusterResource.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(clusterResource).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the clusterResource and deletes it. Returns an error if one occurs.
func (c *clusterResources) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("clusterresources").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *clusterResources) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Resource("clusterresources").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched clusterResource.
func (c *clusterResources) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha2.ClusterResource, err error) {
	result = &v1alpha2.ClusterResource{}
	err = c.client.Patch(pt).
		Resource("clusterresources").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

// Apply takes the given apply declarative configuration, applies it and returns the applied clusterResource.
func (c *clusterResources) Apply(ctx context.Context, clusterResource *samplev1alpha2.ClusterResourceApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha2.ClusterResource, err error) {
	if clusterResource == nil {
		return nil, fmt.Errorf("clusterResource provided to Apply must not be nil")
	}
	patchOpts := opts.ToPatchOptions()
	data, err := json.Marshal(clusterResource)
	if err != nil {
		return nil, err
	}
	name := clusterResource.Name
	if name == nil {
		return nil, fmt.Errorf("clusterResource.Name must be provided to Apply")
	}
	result = &v1alpha2.ClusterResource{}
	err = c.client.Patch(types.ApplyPatchType).
		Resource("clusterresources").
		Name(*name).
		VersionedParams(&patchOpts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

// ApplyStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating ApplyStatus().
func (c *clusterResources) ApplyStatus(ctx context.Context, clusterResource *samplev1alpha2.ClusterResourceApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha2.ClusterResource, err error) {
	if clusterResource == nil {
		return nil, fmt.Errorf("clusterResource provided to Apply must not be nil")
	}
	patchOpts := opts.ToPatchOptions()
	data, err := json.Marshal(clusterResource)
	if err != nil {
		return nil, err
	}

	name := clusterResource.Name
	if name == nil {
		return nil, fmt.Errorf("clusterResource.Name must be provided to Apply")
	}

	result = &v1alpha2.ClusterResource{}
	err = c.client.Patch(types.ApplyPatchType).
		Resource("clusterresources").
		Name(*name).
		SubResource("status").
		VersionedParams(&patchOpts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
