package main

import (
	"flag"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
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

	// ratelimitqueue
	queue := workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[string]())

	// 对 deployment 进行监听
	deployInformer := informerFactory.Apps().V1().Deployments()
	deployInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			fmt.Println("deployment added")
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			fmt.Println("deployment updated")
			key, err := cache.MetaNamespaceKeyFunc(newObj)
			if err == nil {
				queue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			fmt.Println("deployment deleted")
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
	})

	controller := NewController(queue, deployInformer.Informer().GetIndexer(), deployInformer.Informer())
	stopper := make(chan struct{})
	defer close(stopper)

	// 启动 informer
	informerFactory.Start(stopper)

	// 等待 informer 同步完成
	informerFactory.WaitForCacheSync(stopper)

	// 处理队列事件
	go func() {
		for {
			if !controller.processNextItem() {
				break
			}
		}
	}()
}

type Controller struct {
	indexer  cache.Indexer
	queue    workqueue.TypedRateLimitingInterface[string]
	informer cache.Controller
}

func NewController(queue workqueue.TypedRateLimitingInterface[string], indexer cache.Indexer, informer cache.Controller) *Controller {
	return &Controller{
		indexer:  indexer,
		queue:    queue,
		informer: informer,
	}
}

func (c *Controller) processNextItem() bool {
	item, shutdown := c.queue.Get()
	if shutdown {
		return false
	}
	defer c.queue.Done(item)

	err := c.syncToStdouot(item)
	c.HandleErr(err, item)
	return true
}

func (c *Controller) syncToStdouot(key string) error {
	// 通过 key 直接从 indexer 中获取对象
	obj, exists, err := c.indexer.GetByKey(key)
	if err != nil {
		return err
	}
	if !exists {
		fmt.Printf("Deployment %s is deleted\n", key)
		return nil
	}

	deployment := obj.(*appsv1.Deployment)
	return fmt.Errorf("Syncing deployment %s\n", deployment.Name)
}

func (c *Controller) HandleErr(err error, key string) {
	if err == nil {
		c.queue.Forget(key)
		return
	}
	if c.queue.NumRequeues(key) < 5 {
		c.queue.AddRateLimited(key)
		return
	}
	c.queue.Forget(key)
	fmt.Printf("Dropping deployment %q out of the queue: %v\n", key, err)
}
