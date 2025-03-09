package drainnode

import (
	"context"

	"github.com/zhangchl007/autodrainnode/pkg/drainnode"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

// WatchNodes watches for node status changes
func WatchNodes(clientset *kubernetes.Clientset) {
	// Watch for node events
	watcher, err := clientset.CoreV1().Nodes().Watch(context.TODO(), metav1.ListOptions{})
	if err != nil {
		klog.Fatalf("Error watching nodes: %v", err)
	}
	klog.Info("Watching node status changes...")

	// Also watch for specific events related to nodes
	eventWatcher, err := clientset.CoreV1().Events("").Watch(context.TODO(), metav1.ListOptions{})
	if err != nil {
		klog.Fatalf("Error watching events: %v", err)
	}
	klog.Info("Watching for NodeNotReady and Shutdown events...")

	// Handle node status changes
	go watchNodeStatus(clientset, watcher)

	// Handle node events
	watchNodeEvents(clientset, eventWatcher)
}

func watchNodeStatus(clientset *kubernetes.Clientset, watcher watch.Interface) {
	defer watcher.Stop()

	for event := range watcher.ResultChan() {
		node, ok := event.Object.(*v1.Node)
		if !ok {
			continue
		}
		nodeName := node.Name

		switch event.Type {
		case watch.Modified:
			for _, condition := range node.Status.Conditions {
				if condition.Type == v1.NodeReady {
					if condition.Status == v1.ConditionFalse || condition.Status == v1.ConditionUnknown {
						klog.Infof("Node %s is not ready, draining...", nodeName)
						drainnode.DrainNode(clientset, nodeName)
					} else if condition.Status == v1.ConditionTrue {
						klog.Infof("Node %s is back online, uncordoning...", nodeName)
						drainnode.UncordonNode(clientset, nodeName)
					}
				}
			}
		}
	}
}

func watchNodeEvents(clientset *kubernetes.Clientset, eventWatcher watch.Interface) {
	defer eventWatcher.Stop()

	for event := range eventWatcher.ResultChan() {
		ev, ok := event.Object.(*v1.Event)
		if !ok {
			continue
		}

		// Handle both Shutdown and NodeNotReady events
		if ev.InvolvedObject.Kind == "Node" && (ev.Reason == "Shutdown" || ev.Reason == "NodeNotReady") {
			nodeName := ev.InvolvedObject.Name
			klog.Infof("Detected %s event for node %s, starting drain process", ev.Reason, nodeName)
			drainnode.DrainNode(clientset, nodeName)
		}
	}
}
