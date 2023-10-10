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

// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	time "time"

	samplev1alpha1 "github.com/wjiec/programming_k8s/sample_controller/pkg/apis/sample/v1alpha1"
	versioned "github.com/wjiec/programming_k8s/sample_controller/pkg/clientset/versioned"
	internalinterfaces "github.com/wjiec/programming_k8s/sample_controller/pkg/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/wjiec/programming_k8s/sample_controller/pkg/listers/sample/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// ClusterResourceInformer provides access to a shared informer and lister for
// ClusterResources.
type ClusterResourceInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.ClusterResourceLister
}

type clusterResourceInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewClusterResourceInformer constructs a new informer for ClusterResource type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewClusterResourceInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredClusterResourceInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredClusterResourceInformer constructs a new informer for ClusterResource type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredClusterResourceInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.SampleV1alpha1().ClusterResources().List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.SampleV1alpha1().ClusterResources().Watch(context.TODO(), options)
			},
		},
		&samplev1alpha1.ClusterResource{},
		resyncPeriod,
		indexers,
	)
}

func (f *clusterResourceInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredClusterResourceInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *clusterResourceInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&samplev1alpha1.ClusterResource{}, f.defaultInformer)
}

func (f *clusterResourceInformer) Lister() v1alpha1.ClusterResourceLister {
	return v1alpha1.NewClusterResourceLister(f.Informer().GetIndexer())
}
