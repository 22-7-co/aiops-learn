package main

import (
	"flag"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
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

	informerFactory := informers.NewSharedInformerFactory(clientSet, time.Hour*12)

	// Deployment
	deployInformer := informerFactory.Apps().V1().Deployments()
	deployInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			fmt.Println("deployment added")
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			fmt.Println("deployment updated")
		},
		DeleteFunc: func(obj interface{}) {
			fmt.Printf("deployment deleted")
		},
	})

	// service
	serviceInformer := informerFactory.Core().V1().Services()
	serviceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			fmt.Println("service added")
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			fmt.Println("service updated")
		},
		DeleteFunc: func(obj interface{}) {
			fmt.Println("service deleted")
		},
	})
	stopper := make(chan struct{})
	defer close(stopper)

	// 启动 informer,  List &  Watch
	informerFactory.Start(stopper)

	// 等待 informer 同步完成
	informerFactory.WaitForCacheSync(stopper)

	deployments, err := deployInformer.Lister().List(labels.Everything())
	if err != nil {
		panic(err.Error())
	}
	for idx, deploy := range deployments {
		fmt.Println("deployment:\t ", idx, deploy.GetName())
	}

	// 运行 informer
	<-stopper

	fmt.Println("informer stopped")
}
