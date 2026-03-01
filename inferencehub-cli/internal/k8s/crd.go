package k8s

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// CRDExists checks if a Custom Resource Definition exists
func (c *Client) CRDExists(ctx context.Context, crdName string) (bool, error) {
	_, err := c.ApiextensionsClientset.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, crdName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// GetCustomResource retrieves a custom resource
func (c *Client) GetCustomResource(ctx context.Context, gvr schema.GroupVersionResource, namespace, name string) (interface{}, error) {
	var res interface{}
	var err error

	if namespace == "" {
		res, err = c.DynamicClient.Resource(gvr).Get(ctx, name, metav1.GetOptions{})
	} else {
		res, err = c.DynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	}

	if err != nil {
		return nil, err
	}

	return res, nil
}

// CustomResourceExists checks if a custom resource exists
func (c *Client) CustomResourceExists(ctx context.Context, gvr schema.GroupVersionResource, namespace, name string) (bool, error) {
	_, err := c.GetCustomResource(ctx, gvr, namespace, name)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Helper functions for common CRDs

// GatewayGVR returns the GroupVersionResource for Gateway API
func GatewayGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "gateway.networking.k8s.io",
		Version:  "v1",
		Resource: "gateways",
	}
}

// HTTPRouteGVR returns the GroupVersionResource for HTTPRoute
func HTTPRouteGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "gateway.networking.k8s.io",
		Version:  "v1",
		Resource: "httproutes",
	}
}

// GatewayClassGVR returns the GroupVersionResource for GatewayClass
func GatewayClassGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "gateway.networking.k8s.io",
		Version:  "v1",
		Resource: "gatewayclasses",
	}
}

// CertificateGVR returns the GroupVersionResource for cert-manager Certificate
func CertificateGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "cert-manager.io",
		Version:  "v1",
		Resource: "certificates",
	}
}

// ClusterIssuerGVR returns the GroupVersionResource for cert-manager ClusterIssuer
func ClusterIssuerGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "cert-manager.io",
		Version:  "v1",
		Resource: "clusterissuers",
	}
}
