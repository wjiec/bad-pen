package main

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"

	"github.com/wjiec/programming_k8s/machinery"
	"github.com/wjiec/programming_k8s/sample-controller/pkg/clientset/versioned"
	"github.com/wjiec/programming_k8s/sample-controller/pkg/informers/externalversions"
)

func main() {
	config, err := machinery.LoadConfig()
	if err != nil {
		panic(err)
	}

	clientset, err := versioned.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	factory := externalversions.NewSharedInformerFactory(clientset, time.Minute)
	clusterResourceLister := factory.Sample().V1alpha1().ClusterResources().Lister()
	clusterResourceInformer := factory.Sample().V1alpha1().ClusterResources().Informer()
	namespacedResourceLister := factory.Sample().V1alpha2().NamespacedResources().Lister()
	namespacedResourceInformer := factory.Sample().V1alpha2().NamespacedResources().Informer()

	factory.Start(wait.NeverStop)
	if !cache.WaitForCacheSync(wait.NeverStop, clusterResourceInformer.HasSynced, namespacedResourceInformer.HasSynced) {
		runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}

	crs, err := clusterResourceLister.List(labels.Everything())
	if err != nil {
		runtime.HandleError(err)
		return
	}
	fmt.Printf("There are %d ClusterResource in the cluster\n", len(crs))

	nrs, err := namespacedResourceLister.List(labels.Everything())
	if err != nil {
		runtime.HandleError(err)
		return
	}
	fmt.Printf("There are %d NamespacedResource in the cluster\n", len(nrs))
}
