package helm

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
)

// Client wraps Helm functionality
type Client struct {
	config       *action.Configuration
	settings     *cli.EnvSettings
	namespace    string
	debug        bool
}

// NewClient creates a new Helm client
func NewClient(kubeconfigPath, namespace string, debug bool) (*Client, error) {
	settings := cli.New()

	if kubeconfigPath != "" {
		settings.KubeConfig = kubeconfigPath
	}

	// Create action configuration
	actionConfig := new(action.Configuration)

	// Initialize with namespace
	debugLog := func(format string, v ...interface{}) {
		if debug {
			log.Printf(format, v...)
		}
	}

	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, "secret", debugLog); err != nil {
		return nil, fmt.Errorf("failed to initialize Helm client: %w", err)
	}

	// Configure OCI registry client
	registryClient, err := registry.NewClient(
		registry.ClientOptDebug(debug),
		registry.ClientOptWriter(os.Stdout),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create registry client: %w", err)
	}
	actionConfig.RegistryClient = registryClient

	return &Client{
		config:    actionConfig,
		settings:  settings,
		namespace: namespace,
		debug:     debug,
	}, nil
}

// Install installs a Helm chart
func (c *Client) Install(ctx context.Context, releaseName, chartPath string, values map[string]interface{}) (*release.Release, error) {
	client := action.NewInstall(c.config)
	client.Namespace = c.namespace
	client.ReleaseName = releaseName
	client.CreateNamespace = true
	client.Wait = true
	client.Timeout = 600 * 1000000000 // 10 minutes in nanoseconds

	// Load chart
	chart, err := loader.Load(chartPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load chart: %w", err)
	}

	// Install
	rel, err := client.RunWithContext(ctx, chart, values)
	if err != nil {
		return nil, fmt.Errorf("failed to install chart: %w", err)
	}

	return rel, nil
}

// Upgrade upgrades a Helm release
func (c *Client) Upgrade(ctx context.Context, releaseName, chartPath string, values map[string]interface{}) (*release.Release, error) {
	client := action.NewUpgrade(c.config)
	client.Namespace = c.namespace
	client.Wait = true
	client.Timeout = 600 * 1000000000 // 10 minutes
	// Don't reuse values - we provide complete values

	// Load chart
	chart, err := loader.Load(chartPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load chart: %w", err)
	}

	// Upgrade
	rel, err := client.RunWithContext(ctx, releaseName, chart, values)
	if err != nil {
		return nil, fmt.Errorf("failed to upgrade chart: %w", err)
	}

	return rel, nil
}

// Get retrieves a release
func (c *Client) Get(ctx context.Context, releaseName string) (*release.Release, error) {
	client := action.NewGet(c.config)

	rel, err := client.Run(releaseName)
	if err != nil {
		return nil, err
	}

	return rel, nil
}

// List lists all releases in the namespace
func (c *Client) List(ctx context.Context) ([]*release.Release, error) {
	client := action.NewList(c.config)

	releases, err := client.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to list releases: %w", err)
	}

	return releases, nil
}

// Uninstall uninstalls a release
func (c *Client) Uninstall(ctx context.Context, releaseName string) error {
	client := action.NewUninstall(c.config)
	client.Wait = true
	client.Timeout = 300 * 1000000000 // 5 minutes

	_, err := client.Run(releaseName)
	if err != nil {
		return fmt.Errorf("failed to uninstall release: %w", err)
	}

	return nil
}

// ReleaseExists checks if a release exists
func (c *Client) ReleaseExists(ctx context.Context, releaseName string) (bool, error) {
	_, err := c.Get(ctx, releaseName)
	if err != nil {
		if err.Error() == "release: not found" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// AddRepository adds a Helm repository
func (c *Client) AddRepository(name, url string) error {
	repoFile := c.settings.RepositoryConfig

	// Create new repo entry
	entry := &repo.Entry{
		Name: name,
		URL:  url,
	}

	// Create getter providers (enables HTTPS support)
	getters := getter.All(c.settings)

	// Create/update repo file
	r, err := repo.NewChartRepository(entry, getters)
	if err != nil {
		return fmt.Errorf("failed to create chart repository: %w", err)
	}

	// Download index
	if _, err := r.DownloadIndexFile(); err != nil {
		return fmt.Errorf("failed to download index: %w", err)
	}

	// Load existing repos
	f, err := repo.LoadFile(repoFile)
	if os.IsNotExist(err) {
		f = repo.NewFile()
	} else if err != nil {
		return fmt.Errorf("failed to load repository file: %w", err)
	}

	// Add or update
	f.Update(entry)

	// Save
	if err := f.WriteFile(repoFile, 0644); err != nil {
		return fmt.Errorf("failed to write repository file: %w", err)
	}

	return nil
}

// UpdateRepositories updates all configured repositories
func (c *Client) UpdateRepositories() error {
	repoFile := c.settings.RepositoryConfig

	f, err := repo.LoadFile(repoFile)
	if err != nil {
		return fmt.Errorf("failed to load repository file: %w", err)
	}

	// Create getter providers (enables HTTPS support)
	getters := getter.All(c.settings)

	for _, cfg := range f.Repositories {
		r, err := repo.NewChartRepository(cfg, getters)
		if err != nil {
			return fmt.Errorf("failed to create chart repository for %s: %w", cfg.Name, err)
		}

		if _, err := r.DownloadIndexFile(); err != nil {
			return fmt.Errorf("failed to update repository %s: %w", cfg.Name, err)
		}
	}

	return nil
}

// InstallFromRepo installs a Helm chart from a repository
func (c *Client) InstallFromRepo(ctx context.Context, releaseName, repoName, repoURL, chartName, chartVersion, namespace string, values map[string]interface{}) (*release.Release, error) {
	// Validate inputs
	if releaseName == "" || repoName == "" || repoURL == "" || chartName == "" || chartVersion == "" || namespace == "" {
		return nil, fmt.Errorf("all parameters (releaseName, repoName, repoURL, chartName, chartVersion, namespace) are required")
	}

	// Create a new action configuration for this specific namespace
	// This is necessary because each install needs to target the correct namespace
	actionConfig := new(action.Configuration)
	debugLog := func(format string, v ...interface{}) {
		if c.debug {
			log.Printf(format, v...)
		}
	}

	if err := actionConfig.Init(c.settings.RESTClientGetter(), namespace, "secret", debugLog); err != nil {
		return nil, fmt.Errorf("failed to initialize Helm client for namespace %s: %w", namespace, err)
	}

	// Check if release already exists in the target namespace
	getClient := action.NewGet(actionConfig)
	existing, err := getClient.Run(releaseName)
	if err == nil && existing != nil {
		// Release exists - this is OK, just return it
		if c.debug {
			log.Printf("Release %s already exists in namespace %s", releaseName, namespace)
		}
		return existing, nil
	}

	// Add repository
	if err := c.AddRepository(repoName, repoURL); err != nil {
		return nil, fmt.Errorf("failed to add Helm repository %s (%s): %w", repoName, repoURL, err)
	}

	// Update repositories to get latest chart versions
	if err := c.UpdateRepositories(); err != nil {
		return nil, fmt.Errorf("failed to update Helm repositories: %w", err)
	}

	// Create install action with the namespace-specific config
	client := action.NewInstall(actionConfig)
	client.Namespace = namespace
	client.ReleaseName = releaseName
	client.CreateNamespace = true
	client.Wait = true
	client.WaitForJobs = true
	client.Timeout = 600 * 1000000000 // 10 minutes in nanoseconds
	client.Version = chartVersion

	// Locate chart
	chartPath, err := client.ChartPathOptions.LocateChart(fmt.Sprintf("%s/%s", repoName, chartName), c.settings)
	if err != nil {
		return nil, fmt.Errorf("failed to locate chart %s/%s version %s: %w", repoName, chartName, chartVersion, err)
	}

	// Load chart
	chart, err := loader.Load(chartPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load chart from %s: %w", chartPath, err)
	}

	// Validate chart
	if chart == nil {
		return nil, fmt.Errorf("loaded chart is nil")
	}
	if chart.Metadata == nil {
		return nil, fmt.Errorf("chart metadata is nil")
	}

	// Install with context
	rel, err := client.RunWithContext(ctx, chart, values)
	if err != nil {
		return nil, fmt.Errorf("failed to install chart %s (version %s) in namespace %s: %w", chartName, chartVersion, namespace, err)
	}

	return rel, nil
}

// UpgradeFromRepo upgrades a Helm release using a chart from a repository
func (c *Client) UpgradeFromRepo(ctx context.Context, releaseName, repoName, repoURL, chartName, chartVersion, namespace string, values map[string]interface{}) (*release.Release, error) {
	if releaseName == "" || repoName == "" || repoURL == "" || chartName == "" || chartVersion == "" || namespace == "" {
		return nil, fmt.Errorf("all parameters (releaseName, repoName, repoURL, chartName, chartVersion, namespace) are required")
	}

	actionConfig := new(action.Configuration)
	debugLog := func(format string, v ...interface{}) {
		if c.debug {
			log.Printf(format, v...)
		}
	}

	if err := actionConfig.Init(c.settings.RESTClientGetter(), namespace, "secret", debugLog); err != nil {
		return nil, fmt.Errorf("failed to initialize Helm client for namespace %s: %w", namespace, err)
	}

	// Add repository and update
	if err := c.AddRepository(repoName, repoURL); err != nil {
		return nil, fmt.Errorf("failed to add Helm repository %s (%s): %w", repoName, repoURL, err)
	}
	if err := c.UpdateRepositories(); err != nil {
		return nil, fmt.Errorf("failed to update Helm repositories: %w", err)
	}

	client := action.NewUpgrade(actionConfig)
	client.Namespace = namespace
	client.Wait = true
	client.WaitForJobs = true
	client.Timeout = 600 * 1000000000 // 10 minutes in nanoseconds
	client.ReuseValues = false
	client.Version = chartVersion

	chartPath, err := client.ChartPathOptions.LocateChart(fmt.Sprintf("%s/%s", repoName, chartName), c.settings)
	if err != nil {
		return nil, fmt.Errorf("failed to locate chart %s/%s version %s: %w", repoName, chartName, chartVersion, err)
	}

	chart, err := loader.Load(chartPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load chart from %s: %w", chartPath, err)
	}

	rel, err := client.RunWithContext(ctx, releaseName, chart, values)
	if err != nil {
		return nil, fmt.Errorf("failed to upgrade chart %s (version %s) in namespace %s: %w", chartName, chartVersion, namespace, err)
	}

	return rel, nil
}

// UpgradeFromOCI upgrades a Helm release using a chart from an OCI registry
func (c *Client) UpgradeFromOCI(ctx context.Context, releaseName, chartURL, chartVersion, namespace string, values map[string]interface{}) (*release.Release, error) {
	if releaseName == "" || chartURL == "" || chartVersion == "" || namespace == "" {
		return nil, fmt.Errorf("all parameters (releaseName, chartURL, chartVersion, namespace) are required")
	}

	if !strings.HasPrefix(chartURL, "oci://") {
		return nil, fmt.Errorf("chartURL must start with 'oci://', got: %s", chartURL)
	}

	actionConfig := new(action.Configuration)
	debugLog := func(format string, v ...interface{}) {
		if c.debug {
			log.Printf(format, v...)
		}
	}

	if err := actionConfig.Init(c.settings.RESTClientGetter(), namespace, "secret", debugLog); err != nil {
		return nil, fmt.Errorf("failed to initialize Helm client for namespace %s: %w", namespace, err)
	}

	registryClient, err := registry.NewClient(
		registry.ClientOptDebug(c.debug),
		registry.ClientOptWriter(os.Stdout),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create registry client: %w", err)
	}
	actionConfig.RegistryClient = registryClient

	client := action.NewUpgrade(actionConfig)
	client.Namespace = namespace
	client.Wait = true
	client.WaitForJobs = true
	client.Timeout = 600 * 1000000000 // 10 minutes in nanoseconds
	client.ReuseValues = false
	client.Version = chartVersion

	chartPath, err := client.ChartPathOptions.LocateChart(chartURL, c.settings)
	if err != nil {
		return nil, fmt.Errorf("failed to locate OCI chart from %s (version %s): %w", chartURL, chartVersion, err)
	}

	chart, err := loader.Load(chartPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load chart from %s: %w", chartPath, err)
	}

	rel, err := client.RunWithContext(ctx, releaseName, chart, values)
	if err != nil {
		return nil, fmt.Errorf("failed to upgrade OCI chart %s (version %s) in namespace %s: %w", chartURL, chartVersion, namespace, err)
	}

	return rel, nil
}

// InstallFromOCI installs a Helm chart from an OCI registry
func (c *Client) InstallFromOCI(ctx context.Context, releaseName, chartURL, chartVersion, namespace string, values map[string]interface{}) (*release.Release, error) {
	// Validate inputs
	if releaseName == "" || chartURL == "" || chartVersion == "" || namespace == "" {
		return nil, fmt.Errorf("all parameters (releaseName, chartURL, chartVersion, namespace) are required")
	}

	// Validate OCI URL format
	if !strings.HasPrefix(chartURL, "oci://") {
		return nil, fmt.Errorf("chartURL must start with 'oci://', got: %s", chartURL)
	}

	// Create a new action configuration for this specific namespace
	actionConfig := new(action.Configuration)
	debugLog := func(format string, v ...interface{}) {
		if c.debug {
			log.Printf(format, v...)
		}
	}

	if err := actionConfig.Init(c.settings.RESTClientGetter(), namespace, "secret", debugLog); err != nil {
		return nil, fmt.Errorf("failed to initialize Helm client for namespace %s: %w", namespace, err)
	}

	// Configure OCI registry client for this action config
	registryClient, err := registry.NewClient(
		registry.ClientOptDebug(c.debug),
		registry.ClientOptWriter(os.Stdout),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create registry client: %w", err)
	}
	actionConfig.RegistryClient = registryClient

	// Check if release already exists in the target namespace
	getClient := action.NewGet(actionConfig)
	existing, err := getClient.Run(releaseName)
	if err == nil && existing != nil {
		// Release exists - this is OK, just return it
		if c.debug {
			log.Printf("Release %s already exists in namespace %s", releaseName, namespace)
		}
		return existing, nil
	}

	// Create install action with the namespace-specific config
	client := action.NewInstall(actionConfig)
	client.Namespace = namespace
	client.ReleaseName = releaseName
	client.CreateNamespace = true
	client.Wait = true
	client.WaitForJobs = true
	client.Timeout = 600 * 1000000000 // 10 minutes in nanoseconds
	client.Version = chartVersion

	// Locate chart from OCI registry
	chartPath, err := client.ChartPathOptions.LocateChart(chartURL, c.settings)
	if err != nil {
		return nil, fmt.Errorf("failed to locate OCI chart from %s (version %s): %w", chartURL, chartVersion, err)
	}

	// Load chart
	chart, err := loader.Load(chartPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load chart from %s: %w", chartPath, err)
	}

	// Validate chart
	if chart == nil {
		return nil, fmt.Errorf("loaded chart is nil")
	}
	if chart.Metadata == nil {
		return nil, fmt.Errorf("chart metadata is nil")
	}

	// Install with context
	rel, err := client.RunWithContext(ctx, chart, values)
	if err != nil {
		return nil, fmt.Errorf("failed to install OCI chart %s (version %s) in namespace %s: %w", chartURL, chartVersion, namespace, err)
	}

	return rel, nil
}
