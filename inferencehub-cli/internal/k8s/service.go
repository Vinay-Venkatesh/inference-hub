package k8s

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetService retrieves a service
func (c *Client) GetService(ctx context.Context, namespace, name string) (*corev1.Service, error) {
	svc, err := c.Clientset.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get service: %w", err)
	}
	return svc, nil
}

// ServiceExists checks if a service exists
func (c *Client) ServiceExists(ctx context.Context, namespace, name string) (bool, error) {
	_, err := c.GetService(ctx, namespace, name)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
