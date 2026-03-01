package k8s

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetPods returns pods matching the label selector
func (c *Client) GetPods(ctx context.Context, namespace, labelSelector string) (*corev1.PodList, error) {
	pods, err := c.Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}
	return pods, nil
}

// IsPodReady checks if a pod is ready
func (c *Client) IsPodReady(pod *corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

// WaitForPodsReady waits for pods matching the label selector to become ready
func (c *Client) WaitForPodsReady(ctx context.Context, namespace, labelSelector string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		pods, err := c.GetPods(ctx, namespace, labelSelector)
		if err != nil {
			return err
		}

		if len(pods.Items) == 0 {
			// No pods found yet, wait
			time.Sleep(2 * time.Second)
			continue
		}

		allReady := true
		for _, pod := range pods.Items {
			if !c.IsPodReady(&pod) {
				allReady = false
				break
			}
		}

		if allReady {
			return nil
		}

		// Check if any pods are in failed state
		for _, pod := range pods.Items {
			if pod.Status.Phase == corev1.PodFailed {
				return fmt.Errorf("pod %s/%s is in failed state", namespace, pod.Name)
			}
		}

		// Wait before next check
		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("timeout waiting for pods to become ready")
}

// GetPodLogs retrieves logs from a pod
func (c *Client) GetPodLogs(ctx context.Context, namespace, podName, containerName string, tailLines int64) (string, error) {
	options := &corev1.PodLogOptions{
		Container: containerName,
		TailLines: &tailLines,
	}

	req := c.Clientset.CoreV1().Pods(namespace).GetLogs(podName, options)
	logs, err := req.Stream(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get pod logs: %w", err)
	}
	defer logs.Close()

	buf := new(strings.Builder)
	_, err = io.Copy(buf, logs)
	if err != nil {
		return "", fmt.Errorf("failed to read pod logs: %w", err)
	}

	return buf.String(), nil
}
