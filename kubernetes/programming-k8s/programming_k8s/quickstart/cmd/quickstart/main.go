package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"path/filepath"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	for {
		pods, err := clientset.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{})
		if err != nil {
			panic(err)
		}
		fmt.Printf("There are %d pods in the cluster.\n", len(pods.Items))

		meta := pods.Items[rand.Intn(len(pods.Items))]
		pod, err := clientset.CoreV1().Pods(meta.Namespace).Get(context.Background(), meta.Name, metav1.GetOptions{})
		if err != nil {
			panic(err)
		}
		fmt.Printf("The pod %q in the namespace %q with labels: %v\n", pod.Name, pod.Namespace, pod.Labels)

		time.Sleep(10 * time.Second)
	}

}
