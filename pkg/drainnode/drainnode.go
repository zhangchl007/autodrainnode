package drainnode

import (
	"context"
	"fmt"
	"time"

	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

// DrainNode drains pods from a node
func DrainNode(clientset *kubernetes.Clientset, nodeName string) {
	klog.Infof("Starting drain process for node %s", nodeName)

	// Check if the node is still ready
	node, err := clientset.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
	if err != nil {
		klog.Warningf("Error getting node %s: %v", nodeName, err)
		return
	}

	// Check if node is already marked as unschedulable
	if node.Spec.Unschedulable {
		klog.Infof("Node %s is already marked as unschedulable, continuing with drain", nodeName)
	} else {
		// Mark the node as unschedulable (cordon)
		node.Spec.Unschedulable = true
		_, err = clientset.CoreV1().Nodes().Update(context.TODO(), node, metav1.UpdateOptions{})
		if err != nil {
			klog.Warningf("Error marking node %s as unschedulable: %v", nodeName, err)
			return
		}
		klog.Infof("Successfully marked node %s as unschedulable", nodeName)
	}

	// List all pods on the node
	podList, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{
		FieldSelector: "spec.nodeName=" + nodeName,
	})
	if err != nil {
		klog.Warningf("Error listing pods on node %s: %v", nodeName, err)
		return
	}

	// List all daemonsets and find their pod names
	dsList, err := clientset.AppsV1().DaemonSets("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		klog.Warningf("Error listing daemonsets: %v", err)
		return
	}

	dsPodNames := make(map[string]bool)
	for _, ds := range dsList.Items {
		for _, pod := range podList.Items {
			for _, owner := range pod.OwnerReferences {
				if owner.Kind == "DaemonSet" && owner.Name == ds.Name {
					dsPodNames[pod.Name] = true
					break
				}
			}
		}
	}

	// Evict non-daemonset pods
	for _, pod := range podList.Items {
		if !dsPodNames[pod.Name] {
			eviction := &policyv1beta1.Eviction{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pod.Name,
					Namespace: pod.Namespace,
				},
			}
			err := clientset.PolicyV1beta1().Evictions(pod.Namespace).Evict(context.TODO(), eviction)
			if err != nil {
				klog.Warningf("Error evicting pod %s/%s: %v", pod.Namespace, pod.Name, err)
			} else {
				klog.Infof("Evicted pod %s/%s", pod.Namespace, pod.Name)
			}
		}
	}

	// Wait for all non-daemonset pods to be gone
	err = waitUntilNoNonDsPods(clientset, nodeName, dsPodNames, 10*time.Minute)
	if err != nil {
		klog.Warningf("Timeout or error waiting for pods to be evicted: %v", err)
	} else {
		klog.Infof("All non-daemonset pods have been evicted from node %s", nodeName)
	}
}

// UncordonNode removes the unschedulable taint from a node
func UncordonNode(clientset *kubernetes.Clientset, nodeName string) error {
	node, err := clientset.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
	if err != nil {
		klog.Warningf("Error getting node %s: %v", nodeName, err)
		return err
	}

	if !node.Spec.Unschedulable {
		klog.Infof("Node %s is already schedulable", nodeName)
		return nil
	}

	node.Spec.Unschedulable = false
	_, err = clientset.CoreV1().Nodes().Update(context.TODO(), node, metav1.UpdateOptions{})
	if err != nil {
		klog.Warningf("Error uncordoning node %s: %v", nodeName, err)
		return err
	}

	klog.Infof("Successfully uncordoned node %s", nodeName)
	return nil
}

func waitUntilNoNonDsPods(clientset *kubernetes.Clientset, nodeName string, dsPodNames map[string]bool, timeout time.Duration) error {
	start := time.Now()
	for {
		if time.Since(start) > timeout {
			return fmt.Errorf("timeout waiting for pods to be evicted on node %s", nodeName)
		}
		podList, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{
			FieldSelector: "spec.nodeName=" + nodeName,
		})
		if err != nil {
			return err
		}
		hasNonDsPods := false
		for _, pod := range podList.Items {
			if !dsPodNames[pod.Name] {
				hasNonDsPods = true
				break
			}
		}
		if !hasNonDsPods {
			return nil
		}
		time.Sleep(5 * time.Second)
	}
}
