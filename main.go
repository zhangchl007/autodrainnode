package main

import (
	"github.com/zhangchl007/autodrainnode/pkg/watchevent"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

func main() {
	// Use in-cluster config since running as a pod
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		klog.Fatalf("Error getting in-cluster config: %v", err)
	}

	// Create the clientset
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		klog.Fatalf("Error creating clientset: %v", err)
	}

	// Start watching for node events
	klog.Info("Starting node watcher")
	watchevent.WatchNodes(clientset)
}
