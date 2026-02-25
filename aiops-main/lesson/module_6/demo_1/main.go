package main

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {

	// demo1
	// // 加载 kubeconfig 配置
	// var kubeconfig *string
	// if home := homedir.HomeDir(); home != "" {
	// 	kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "[可选] kubeconfig 绝对路径")
	// } else {
	// 	kubeconfig = flag.String("kubeconfig", "", "kubeconfig 绝对路径")
	// }
	// config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	// if err != nil {
	// 	fmt.Printf("error %s", err.Error())
	// }

	// demo2 in cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		fmt.Printf("error %s", err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Printf("error %s", err.Error())
	}

	// 可以看一下 Pods 里面有什么操作
	pods, err := clientset.CoreV1().Pods("default").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("error %s", err.Error())
	}
	for _, pod := range pods.Items {
		fmt.Printf("Pod name %s\n", pod.Name)
	}
	// 这里可以运行一下结果
	// 创建一个 pod
	// kubectl create deployment nginx --image nginx

	// 继续
	deployment, err := clientset.AppsV1().Deployments("default").List(context.Background(), metav1.ListOptions{})
	for _, d := range deployment.Items {
		fmt.Printf("deployment name %s", d.Name)
	}
}
