package main

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
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
	// 指定Kubernetes API路径
	config.APIPath = "api"
	// 指定要使用的API组和版本
	config.GroupVersion = &corev1.SchemeGroupVersion
	// 指定编解码器
	config.NegotiatedSerializer = scheme.Codecs

	// 生成RESTClient实例
	restClient, err := rest.RESTClientFor(config)
	if err != nil {
		panic(err)
	}

	// 创建空的结构体，存储pod列表
	podList := &corev1.PodList{}

	// 构建HTTP请求参数，直接定义 GVR
	// Get请求
	restClient.Get().
		// 指定命名空间
		Namespace("kube-system").
		// 指定要获取的资源类型
		Resource("pods").
		// 设置请求参数，使用metav1.ListOptions结构体设置了Limit参数为500，并使用scheme.ParameterCodec进行参数编码。
		VersionedParams(&metav1.ListOptions{Limit: 500}, scheme.ParameterCodec).
		// 发送请求并获取响应，使用context.TODO()作为上下文
		Do(context.TODO()).
		// 将响应解码为podList
		Into(podList)

	for _, v := range podList.Items {
		fmt.Printf("NameSpace: %v  Name: %v  Status: %v \n", v.Namespace, v.Name, v.Status.Phase)
	}
}
