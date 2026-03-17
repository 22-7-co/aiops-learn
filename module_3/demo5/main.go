package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	kubeconfig := flag.String("kubeconfig", "/root/.kube/config", "path to the kubeconfig file")
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	timeout := int64(60)
	watcher, err := clientSet.CoreV1().Pods("default").Watch(context.TODO(), metav1.ListOptions{TimeoutSeconds: &timeout})
	if err != nil {
		panic(err.Error())
	}
	defer watcher.Stop()
	for event := range watcher.ResultChan() {
		item := event.Object.(*corev1.Pod)

		switch event.Type {
		case watch.Added:
			processPod("Added", item.GetName())
		case watch.Deleted:
			processPod("Deleted", item.GetName())
		case watch.Modified:
			processPod("Modified", item.GetName())
		}
	}
}

func processPod(eventType string, name string) {
	fmt.Println("processing pod\t", eventType, "\t", name)
}
