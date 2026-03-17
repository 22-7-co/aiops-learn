package main

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

func main() {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	config.APIPath = "/apis"
	config.GroupVersion = &corev1.SchemeGroupVersion
	config.NegotiatedSerializer = scheme.Codecs

	// 初始化 restclient
	restClient, err := rest.RESTClientFor(config)
	if err != nil {
		panic(err.Error())
	}
	// 获取 pod  列表
	pods := &corev1.PodList{}
	restClient.Get().Namespace("kube-system").Resource("pods").VersionedParams(&metav1.ListOptions{Limit: 500}, scheme.ParameterCodec).Do(context.Background()).Into(pods)
	for _, pod := range pods.Items {
		fmt.Println(pod.Name, "\t", pod.Namespace, "\t", pod.Status.Phase)
	}
}
