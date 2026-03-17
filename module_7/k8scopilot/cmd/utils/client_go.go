package utils

import (
	"os"
	"path/filepath"
	"strings"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type ClientGo struct {
	Clientset       *kubernetes.Clientset
	DynamicClient   dynamic.Interface
	DiscoveryClient discovery.DiscoveryInterface
}

func NewClientGo(kubeconfig string) (*ClientGo, error) {
	if strings.HasPrefix(kubeconfig, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		kubeconfig = filepath.Join(homeDir, kubeconfig[1:])
	}
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, err
	}
	return &ClientGo{
		Clientset:       kubernetes.NewForConfigOrDie(config),
		DynamicClient:   dynamicClient,
		DiscoveryClient: discoveryClient,
	}, nil
}
