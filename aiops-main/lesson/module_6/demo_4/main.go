package main

import (
	"context"
	_ "embed"
	"flag"
	"log"
	"strings"
	"path/filepath"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

//go:embed deploy.yaml
var deployYaml string

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

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	// 使用yaml.Unmarshal()将deployYaml解析为unstructured.Unstructured对象
	deployObj := &unstructured.Unstructured{}
	if err := yaml.Unmarshal([]byte(deployYaml), deployObj); err != nil {
		panic(err)
	}

	// 从deployObj中提取apiVersion和kind以确定GVR
	apiVersion, found, err := unstructured.NestedString(deployObj.Object, "apiVersion")
	if err != nil || !found {
		log.Fatalln("apiVersion not found:", err)
	}

	kind, found, err := unstructured.NestedString(deployObj.Object, "kind")
	if err != nil || !found {
		log.Fatalln("kind not found:", err)
	}

	// 根据apiVersion生成GVR
	gvr := schema.GroupVersionResource{}
	versionParts := strings.Split(apiVersion, "/")
	if len(versionParts) == 2 {
		gvr.Group = versionParts[0]
		gvr.Version = versionParts[1]
	} else {
		// Pod 的话没有 group，只有 version
		gvr.Version = versionParts[0]
	}

	// 根据kind确定资源名称
	switch kind {
	case "Deployment":
		gvr.Resource = "deployments"
	case "Pod":
		gvr.Resource = "pods"
	// 可以根据需要添加更多的kind
	default:
		log.Fatalf("unsupported kind: %s", kind)
	}

	// 使用dynamicClient.Resource()指定命名空间和资源选项, Create()方法创建deployment
	_, err = dynamicClient.Resource(gvr).Namespace("default").Create(context.TODO(), deployObj, v1.CreateOptions{})
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("create deployment success")
}
