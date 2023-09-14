package main

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/wjiec/programming_k8s/machinery"
)

func main() {
	clientset, err := machinery.NewClientset()
	if err != nil {
		panic(err)
	}

	w, err := clientset.CoreV1().Pods("").Watch(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err)
	}
	defer w.Stop()

	for ev := range w.ResultChan() {
		pod := ev.Object.(*corev1.Pod)

		fmt.Printf("Event %q occurred with name is %q in namespace %q\n", ev.Type, pod.Name, pod.Namespace)
	}
}
