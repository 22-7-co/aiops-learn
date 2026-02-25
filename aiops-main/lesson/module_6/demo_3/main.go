package main

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {
	// 加载 kubeconfig 配置
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "[可选] kubeconfig 绝对路径")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "kubeconfig 绝对路径")
	}
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}
	// 根据配置创建dynamicClient
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	// 指定要操作资源的版本和类型
	// 这里直接用 gvr 来指定资源类型
	gvr := schema.GroupVersionResource{
		Version:  "v1",
		Resource: "pods",
	}
	// 获取资源，返回的是*unstructured.UnstructuredList指针类型
	unStructObj, err := dynamicClient.Resource(gvr).Namespace("kube-system").List(
		context.TODO(),
		metav1.ListOptions{},
	)
	if err != nil {
		panic(err)
	}

	podList := &corev1.PodList{}
	// 将unstructured对象转换为podList
	if err = runtime.DefaultUnstructuredConverter.FromUnstructured(unStructObj.UnstructuredContent(), podList); err != nil {
		panic(err)
	}

	for _, v := range podList.Items {
		fmt.Printf("namespace: %s, name: %s, status: %s\n", v.Namespace, v.Name, v.Status.Phase)
	}
}
