package cli

import (
	"context"
	"fmt"

	"github.com/Vinay-Venkatesh/inferencehub-cli/internal/helm"
	"github.com/Vinay-Venkatesh/inferencehub-cli/internal/k8s"
	"github.com/Vinay-Venkatesh/inferencehub-cli/internal/ui"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show installation status",
	Long:  "Display the current status of the InferenceHub installation",
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Create UI
	uiInstance := ui.New(verbose)
	uiInstance.PrintHeader("Installation Status")

	// Create Kubernetes client
	k8sClient, err := k8s.NewClient(getKubeconfigPath())
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	// Verify connection
	uiInstance.Info("Connecting to cluster...")
	if err := k8sClient.VerifyConnection(ctx); err != nil {
		return fmt.Errorf("failed to connect to cluster: %w", err)
	}

	// Get cluster version
	version, _ := k8sClient.GetServerVersion(ctx)

	// Check namespace
	nsExists, err := k8sClient.NamespaceExists(ctx, namespace)
	if err != nil {
		return fmt.Errorf("failed to check namespace: %w", err)
	}

	uiInstance.PrintPhase("Cluster Information")
	uiInstance.PrintKeyValue("Kubernetes Version", version)
	uiInstance.PrintKeyValue("Namespace", namespace)

	if nsExists {
		uiInstance.Success("Namespace exists")
	} else {
		uiInstance.Warn("Namespace does not exist - platform not installed")
		return nil
	}

	// Check Helm release
	uiInstance.PrintPhase("Helm Release")
	helmClient, err := helm.NewClient(getKubeconfigPath(), namespace, verbose)
	if err != nil {
		return fmt.Errorf("failed to create Helm client: %w", err)
	}

	releaseExists, err := helmClient.ReleaseExists(ctx, "inferencehub")
	if err != nil {
		uiInstance.Warn("Could not check Helm release: %v", err)
	} else if releaseExists {
		release, err := helmClient.Get(ctx, "inferencehub")
		if err == nil {
			uiInstance.PrintKeyValue("Release Name", release.Name)
			uiInstance.PrintKeyValue("Chart", fmt.Sprintf("%s-%s", release.Chart.Metadata.Name, release.Chart.Metadata.Version))
			uiInstance.PrintKeyValue("Status", string(release.Info.Status))
			uiInstance.PrintKeyValue("Last Deployed", release.Info.LastDeployed.Format("2006-01-02 15:04:05"))
			uiInstance.PrintKeyValue("Revision", fmt.Sprintf("%d", release.Version))
		}
	} else {
		uiInstance.Warn("Helm release 'inferencehub' not found")
		return nil
	}

	// Check pods
	uiInstance.PrintPhase("Component Status")

	type componentDef struct {
		display  string
		selector string
	}
	componentDefs := []componentDef{
		{"openwebui", "app.kubernetes.io/name=openwebui,app.kubernetes.io/instance=inferencehub"},
		{"litellm", "app.kubernetes.io/name=litellm,app.kubernetes.io/instance=inferencehub"},
		{"postgresql", "app.kubernetes.io/component=database,app.kubernetes.io/instance=inferencehub"},
		{"redis-openwebui", "app.kubernetes.io/component=cache-openwebui,app.kubernetes.io/instance=inferencehub"},
		{"redis-litellm", "app.kubernetes.io/component=cache-litellm,app.kubernetes.io/instance=inferencehub"},
	}

	// Include SearXNG only when it is deployed (web search enabled)
	searxngDeployed, _ := k8sClient.ServiceExists(ctx, namespace, "inferencehub-searxng")
	if searxngDeployed {
		componentDefs = append(componentDefs, componentDef{"searxng", "app.kubernetes.io/component=websearch,app.kubernetes.io/instance=inferencehub"})
	}
	statusItems := []ui.StatusItem{}

	for _, comp := range componentDefs {
		component := comp.display
		pods, err := k8sClient.GetPods(ctx, namespace, comp.selector)

		if err != nil {
			statusItems = append(statusItems, ui.StatusItem{
				Name:    component,
				Message: fmt.Sprintf("Error: %v", err),
				Success: false,
			})
			continue
		}

		if len(pods.Items) == 0 {
			statusItems = append(statusItems, ui.StatusItem{
				Name:    component,
				Message: "No pods found",
				Success: false,
			})
			continue
		}

		readyCount := 0
		totalCount := len(pods.Items)

		for _, pod := range pods.Items {
			if k8sClient.IsPodReady(&pod) {
				readyCount++
			}
		}

		success := readyCount == totalCount
		message := fmt.Sprintf("%d/%d pods ready", readyCount, totalCount)

		if success {
			message += " ✓"
		}

		statusItems = append(statusItems, ui.StatusItem{
			Name:    component,
			Message: message,
			Success: success,
		})
	}

	uiInstance.PrintStatusTable(statusItems)

	// Check services
	uiInstance.PrintPhase("Services")

	services := []string{
		"inferencehub-openwebui",
		"inferencehub-litellm",
		"inferencehub-postgresql",
		"inferencehub-redis-openwebui",
		"inferencehub-redis-litellm",
	}
	if searxngDeployed {
		services = append(services, "inferencehub-searxng")
	}

	svcItems := []ui.StatusItem{}

	for _, svcName := range services {
		exists, err := k8sClient.ServiceExists(ctx, namespace, svcName)

		if err != nil {
			svcItems = append(svcItems, ui.StatusItem{
				Name:    svcName,
				Message: fmt.Sprintf("Error: %v", err),
				Success: false,
			})
		} else if exists {
			svcItems = append(svcItems, ui.StatusItem{
				Name:    svcName,
				Message: "Ready",
				Success: true,
			})
		} else {
			svcItems = append(svcItems, ui.StatusItem{
				Name:    svcName,
				Message: "Not found",
				Success: false,
			})
		}
	}

	uiInstance.PrintStatusTable(svcItems)

	// Summary
	uiInstance.PrintSeparator()

	allHealthy := true
	for _, item := range statusItems {
		if !item.Success {
			allHealthy = false
			break
		}
	}

	if allHealthy {
		uiInstance.Success("All components are healthy!")
	} else {
		uiInstance.Warn("Some components are not healthy")
		uiInstance.Info("Run 'inferencehub verify' for detailed diagnostics")
	}

	return nil
}
