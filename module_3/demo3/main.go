package main

import (
	"context"
	"flag"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func main() {
	kubeconfig := flag.String("kubeconfig", "/root/.kube/config", "path to the kubeconfig file")
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	deploymentGVR := schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "pods",
	}

	unStructPodList, err := dynamicClient.Resource(deploymentGVR).Namespace("kube-system").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	// 定义一个 PodList Struct
	podList := &corev1.PodList{}
	// 将 Unstructured 转换为 PodList
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unStructPodList.UnstructuredContent(), podList)
	if err != nil {
		panic(err.Error())
	}
	for _, pod := range podList.Items {
		fmt.Println(pod.Name, "\t", pod.Namespace, "\t", pod.Status.Phase)
	}
}
