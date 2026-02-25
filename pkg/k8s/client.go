package k8s

import (
	"fmt"
	"log/slog"
	"path/filepath"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// Clients holds all Kubernetes client interfaces.
type Clients struct {
	Clientset     kubernetes.Interface
	DynamicClient dynamic.Interface
	Discovery     discovery.DiscoveryInterface
}

// NewClients creates Kubernetes clients, trying in-cluster config first,
// then falling back to kubeconfig.
func NewClients() (*Clients, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		slog.Info("in-cluster config not available, falling back to kubeconfig", "error", err)
		kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
		cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to build kubernetes config: %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	dynClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	disc, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery client: %w", err)
	}

	return &Clients{
		Clientset:     clientset,
		DynamicClient: dynClient,
		Discovery:     disc,
	}, nil
}
