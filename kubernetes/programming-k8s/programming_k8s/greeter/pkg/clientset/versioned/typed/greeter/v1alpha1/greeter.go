/*
Copyright The Kubernetes Authors.

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
// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	json "encoding/json"
	"fmt"
	"time"

	v1alpha1 "github.com/wjiec/programming_k8s/greeter/pkg/apis/greeter/v1alpha1"
	greeterv1alpha1 "github.com/wjiec/programming_k8s/greeter/pkg/applyconfiguration/greeter/v1alpha1"
	scheme "github.com/wjiec/programming_k8s/greeter/pkg/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// GreetersGetter has a method to return a GreeterInterface.
// A group's client should implement this interface.
type GreetersGetter interface {
	Greeters(namespace string) GreeterInterface
}

// GreeterInterface has methods to work with Greeter resources.
type GreeterInterface interface {
	Create(ctx context.Context, greeter *v1alpha1.Greeter, opts v1.CreateOptions) (*v1alpha1.Greeter, error)
	Update(ctx context.Context, greeter *v1alpha1.Greeter, opts v1.UpdateOptions) (*v1alpha1.Greeter, error)
	UpdateStatus(ctx context.Context, greeter *v1alpha1.Greeter, opts v1.UpdateOptions) (*v1alpha1.Greeter, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.Greeter, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.GreeterList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.Greeter, err error)
	Apply(ctx context.Context, greeter *greeterv1alpha1.GreeterApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.Greeter, err error)
	ApplyStatus(ctx context.Context, greeter *greeterv1alpha1.GreeterApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.Greeter, err error)
	GreeterExpansion
}

// greeters implements GreeterInterface
type greeters struct {
	client rest.Interface
	ns     string
}

// newGreeters returns a Greeters
func newGreeters(c *GreeterV1alpha1Client, namespace string) *greeters {
	return &greeters{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the greeter, and returns the corresponding greeter object, and an error if there is any.
func (c *greeters) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.Greeter, err error) {
	result = &v1alpha1.Greeter{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("greeters").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Greeters that match those selectors.
func (c *greeters) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.GreeterList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.GreeterList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("greeters").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested greeters.
func (c *greeters) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("greeters").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a greeter and creates it.  Returns the server's representation of the greeter, and an error, if there is any.
func (c *greeters) Create(ctx context.Context, greeter *v1alpha1.Greeter, opts v1.CreateOptions) (result *v1alpha1.Greeter, err error) {
	result = &v1alpha1.Greeter{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("greeters").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(greeter).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a greeter and updates it. Returns the server's representation of the greeter, and an error, if there is any.
func (c *greeters) Update(ctx context.Context, greeter *v1alpha1.Greeter, opts v1.UpdateOptions) (result *v1alpha1.Greeter, err error) {
	result = &v1alpha1.Greeter{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("greeters").
		Name(greeter.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(greeter).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *greeters) UpdateStatus(ctx context.Context, greeter *v1alpha1.Greeter, opts v1.UpdateOptions) (result *v1alpha1.Greeter, err error) {
	result = &v1alpha1.Greeter{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("greeters").
		Name(greeter.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(greeter).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the greeter and deletes it. Returns an error if one occurs.
func (c *greeters) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("greeters").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *greeters) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("greeters").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched greeter.
func (c *greeters) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.Greeter, err error) {
	result = &v1alpha1.Greeter{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("greeters").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

// Apply takes the given apply declarative configuration, applies it and returns the applied greeter.
func (c *greeters) Apply(ctx context.Context, greeter *greeterv1alpha1.GreeterApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.Greeter, err error) {
	if greeter == nil {
		return nil, fmt.Errorf("greeter provided to Apply must not be nil")
	}
	patchOpts := opts.ToPatchOptions()
	data, err := json.Marshal(greeter)
	if err != nil {
		return nil, err
	}
	name := greeter.Name
	if name == nil {
		return nil, fmt.Errorf("greeter.Name must be provided to Apply")
	}
	result = &v1alpha1.Greeter{}
	err = c.client.Patch(types.ApplyPatchType).
		Namespace(c.ns).
		Resource("greeters").
		Name(*name).
		VersionedParams(&patchOpts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

// ApplyStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating ApplyStatus().
func (c *greeters) ApplyStatus(ctx context.Context, greeter *greeterv1alpha1.GreeterApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.Greeter, err error) {
	if greeter == nil {
		return nil, fmt.Errorf("greeter provided to Apply must not be nil")
	}
	patchOpts := opts.ToPatchOptions()
	data, err := json.Marshal(greeter)
	if err != nil {
		return nil, err
	}

	name := greeter.Name
	if name == nil {
		return nil, fmt.Errorf("greeter.Name must be provided to Apply")
	}

	result = &v1alpha1.Greeter{}
	err = c.client.Patch(types.ApplyPatchType).
		Namespace(c.ns).
		Resource("greeters").
		Name(*name).
		SubResource("status").
		VersionedParams(&patchOpts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
