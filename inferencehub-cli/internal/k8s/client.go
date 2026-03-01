package k8s

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

// Client wraps Kubernetes client functionality
type Client struct {
	Clientset              *kubernetes.Clientset
	DynamicClient          dynamic.Interface
	ApiextensionsClientset *apiextensionsclientset.Clientset
	Config                 *rest.Config
}

// NewClient creates a new Kubernetes client
func NewClient(kubeconfigPath string) (*Client, error) {
	// Use default kubeconfig path if not provided
	if kubeconfigPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
		kubeconfigPath = filepath.Join(home, ".kube", "config")
	}

	// Build config from kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build kubeconfig: %w", err)
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	// Create dynamic client for CRDs
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	// Create apiextensions client for CRD operations
	apiextensionsClient, err := apiextensionsclientset.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create apiextensions client: %w", err)
	}

	return &Client{
		Clientset:              clientset,
		DynamicClient:          dynamicClient,
		ApiextensionsClientset: apiextensionsClient,
		Config:                 config,
	}, nil
}

// VerifyConnection verifies connectivity to the Kubernetes cluster
func (c *Client) VerifyConnection(ctx context.Context) error {
	_, err := c.Clientset.Discovery().ServerVersion()
	if err != nil {
		return fmt.Errorf("failed to connect to Kubernetes cluster: %w", err)
	}
	return nil
}

// GetServerVersion returns the Kubernetes server version
func (c *Client) GetServerVersion(ctx context.Context) (string, error) {
	version, err := c.Clientset.Discovery().ServerVersion()
	if err != nil {
		return "", err
	}
	return version.String(), nil
}

// NamespaceExists checks if a namespace exists
func (c *Client) NamespaceExists(ctx context.Context, namespace string) (bool, error) {
	_, err := c.Clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// CreateNamespace creates a namespace
func (c *Client) CreateNamespace(ctx context.Context, namespace string) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}

	_, err := c.Clientset.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	return nil
}

// GetAvailableStorageClasses returns a list of available storage classes
func (c *Client) GetAvailableStorageClasses(ctx context.Context) ([]string, error) {
	scList, err := c.Clientset.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list storage classes: %w", err)
	}

	var storageClasses []string
	for _, sc := range scList.Items {
		storageClasses = append(storageClasses, sc.Name)
	}

	return storageClasses, nil
}

// GetDefaultStorageClass returns the default storage class or a suitable fallback
func (c *Client) GetDefaultStorageClass(ctx context.Context) (string, error) {
	scList, err := c.Clientset.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list storage classes: %w", err)
	}

	if len(scList.Items) == 0 {
		return "", fmt.Errorf("no storage classes found in cluster")
	}

	// Priority order for selecting storage class
	// 1. Check for default storage class annotation
	for _, sc := range scList.Items {
		if sc.Annotations["storageclass.kubernetes.io/is-default-class"] == "true" ||
			sc.Annotations["storageclass.beta.kubernetes.io/is-default-class"] == "true" {
			return sc.Name, nil
		}
	}

	// 2. Look for common AWS storage classes (gp3 > gp2 > standard)
	preferredSC := []string{"gp3", "gp2", "standard"}
	for _, preferred := range preferredSC {
		for _, sc := range scList.Items {
			if sc.Name == preferred {
				return sc.Name, nil
			}
		}
	}

	// 3. Return the first available storage class
	return scList.Items[0].Name, nil
}

// StorageClassExists checks if a storage class exists
func (c *Client) StorageClassExists(ctx context.Context, name string) (bool, error) {
	_, err := c.Clientset.StorageV1().StorageClasses().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ApplyManifestFromURL fetches and applies Kubernetes manifests from a URL
func (c *Client) ApplyManifestFromURL(ctx context.Context, url string) error {
	// Fetch the manifest from URL
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch manifest from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch manifest: HTTP %d", resp.StatusCode)
	}

	// Read the response body
	manifestBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	// Apply the manifest
	return c.ApplyManifest(ctx, manifestBytes)
}

// ApplyManifest applies Kubernetes manifests from YAML bytes
func (c *Client) ApplyManifest(ctx context.Context, manifestBytes []byte) error {
	// Create a REST mapper to discover resource types
	discoveryClient := c.Clientset.Discovery()
	groupResources, err := restmapper.GetAPIGroupResources(discoveryClient)
	if err != nil {
		return fmt.Errorf("failed to get API group resources: %w", err)
	}
	mapper := restmapper.NewDiscoveryRESTMapper(groupResources)

	// Split the YAML into individual documents
	decoder := yaml.NewYAMLOrJSONDecoder(io.NopCloser(io.Reader(NewBytesReader(manifestBytes))), 4096)

	for {
		// Decode one object at a time
		var rawObj map[string]interface{}
		if err := decoder.Decode(&rawObj); err != nil {
			if err == io.EOF {
				break // No more objects
			}
			return fmt.Errorf("failed to decode manifest: %w", err)
		}

		if rawObj == nil {
			continue
		}

		// Convert to unstructured object
		obj := &unstructured.Unstructured{Object: rawObj}

		// Get GVK (GroupVersionKind) from the object
		gvk := obj.GroupVersionKind()
		if gvk.Empty() {
			continue // Skip objects without GVK
		}

		// Map GVK to GVR (GroupVersionResource)
		mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return fmt.Errorf("failed to get REST mapping for %s: %w", gvk.String(), err)
		}

		// Determine if this is a namespaced resource
		var dr dynamic.ResourceInterface
		if mapping.Scope.Name() == "namespace" {
			// Namespaced resource
			namespace := obj.GetNamespace()
			if namespace == "" {
				namespace = "default"
			}
			dr = c.DynamicClient.Resource(mapping.Resource).Namespace(namespace)
		} else {
			// Cluster-scoped resource
			dr = c.DynamicClient.Resource(mapping.Resource)
		}

		// Try to apply (create or update)
		_, err = dr.Apply(ctx, obj.GetName(), obj, metav1.ApplyOptions{
			FieldManager: "inferencehub-cli",
			Force:        true,
		})

		if err != nil {
			return fmt.Errorf("failed to apply %s %s: %w", gvk.Kind, obj.GetName(), err)
		}
	}

	return nil
}

// NewBytesReader creates a new bytes reader from bytes
func NewBytesReader(b []byte) io.Reader {
	return &bytesReader{data: b, index: 0}
}

type bytesReader struct {
	data  []byte
	index int
}

func (r *bytesReader) Read(p []byte) (n int, err error) {
	if r.index >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.index:])
	r.index += n
	return n, nil
}
