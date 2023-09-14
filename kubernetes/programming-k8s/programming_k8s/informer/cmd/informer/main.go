package main

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"

	"github.com/wjiec/programming_k8s/machinery"
)

func main() {
	clientset, err := machinery.NewClientset()
	if err != nil {
		panic(err)
	}
	defer runtime.HandleCrash()

	factory := informers.NewSharedInformerFactory(clientset, time.Minute)
	podInformer := factory.Core().V1().Pods()

	factory.Start(wait.NeverStop)
	if !cache.WaitForCacheSync(wait.NeverStop, podInformer.Informer().HasSynced) {
		runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}

	_, err = podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			pod := obj.(*corev1.Pod)

			fmt.Printf("pod %q has been add into %q namespace in the cluster\n", pod.Name, pod.Namespace)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			pod := newObj.(*corev1.Pod)

			fmt.Printf("pod %q in the %q namespace has been modified\n", pod.Name, pod.Namespace)
		},
		DeleteFunc: func(obj interface{}) {
			pod := obj.(*corev1.Pod)

			fmt.Printf("pod %q has been deleted from %q namespace\n", pod.Name, pod.Namespace)
		},
	})
	if err != nil {
		runtime.HandleError(err)
		return
	}

	for {
		pods, err := podInformer.Lister().List(labels.Everything())
		if err != nil {
			runtime.HandleError(err)
			return
		}
		fmt.Printf("There are %d pods in the cluster\n", len(pods))

		time.Sleep(10 * time.Second)
	}
}
