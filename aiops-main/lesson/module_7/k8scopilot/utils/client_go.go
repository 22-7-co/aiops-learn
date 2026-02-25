package utils

import (
	"fmt"
	"path/filepath"
	"strings"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// ClientGo encapsulates both clientset and dynamicClient for Kubernetes operations
type ClientGo struct {
	Clientset       *kubernetes.Clientset
	DynamicClient   dynamic.Interface
	DiscoveryClient discovery.DiscoveryInterface
}

// NewClientGo initializes a new ClientGo instance with the provided kubeconfig path
func NewClientGo(kubeconfig string) (*ClientGo, error) {
	// Handle ~ in the kubeconfig path
	if strings.HasPrefix(kubeconfig, "~") {
		homeDir := homedir.HomeDir()
		kubeconfig = filepath.Join(homeDir, kubeconfig[1:])
	}

	// Build the configuration from the kubeconfig file
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build kubeconfig: %w", err)
	}

	// Create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	// Create the dynamic client
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	// Create DiscoveryClient
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery client: %w", err)
	}

	return &ClientGo{
		Clientset:       clientset,
		DynamicClient:   dynamicClient,
		DiscoveryClient: discoveryClient,
	}, nil
}

// Example usage in another module:
//
// func main() {
//     clientGo, err := utils.NewClientGo("~/.kube/config")
//     if err != nil {
//         fmt.Println("Error creating Kubernetes clients:", err)
//         os.Exit(1)
//     }
//     // Use clientGo.Clientset and clientGo.DynamicClient as needed...
// }
