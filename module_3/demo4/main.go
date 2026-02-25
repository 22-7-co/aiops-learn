package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"strings"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

//go:embed deploy.yaml
var deployYaml string

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
	// 解析 YAML 转成 Unstructured
	deployObj := &unstructured.Unstructured{}
	if err := yaml.Unmarshal([]byte(deployYaml), deployObj); err != nil {
		fmt.Printf("err %s", err.Error())
	}
	// 从 deployObj 中获取 Deployment 的 GroupVersionResource
	apiVersion, found, err := unstructured.NestedString(deployObj.Object, "apiVersion")
	if err != nil || !found {
		fmt.Printf("err %s", err.Error())
	}
	kind, found, err := unstructured.NestedString(deployObj.Object, "kind")
	if err != nil || !found {
		fmt.Printf("err %s", err.Error())
	}

	//  指定 GVR
	gvr := schema.GroupVersionResource{}
	versionParts := strings.Split(apiVersion, "/")
	if len(versionParts) == 2 {
		gvr.Group = versionParts[0]   // apps  或  core
		gvr.Version = versionParts[1] // v1
	} else { //   v1  == core/v1
		gvr.Version = versionParts[0] // v1
	}
	switch kind {
	case "Deployment":
		gvr.Resource = "deployments"
	case "Pod":
		gvr.Resource = "pods"
	default:
		fmt.Printf("Unsupported kind %s", kind)
	}

	// 使用 dynamicClient 创建 资源
	_, err = dynamicClient.Resource(gvr).Namespace("default").Create(context.TODO(), deployObj, v1.CreateOptions{})
	if err != nil {
		fmt.Printf("err %s", err.Error())
	}
	fmt.Printf("create resource %s successfully\n", deployObj.GetName())
}
