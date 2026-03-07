package cli

import (
	"context"
	"fmt"

	"github.com/Vinay-Venkatesh/inferencehub-cli/internal/k8s"
	"github.com/Vinay-Venkatesh/inferencehub-cli/internal/ui"
	"github.com/spf13/cobra"
)

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify prerequisites and installation",
	Long:  "Check infrastructure prerequisites and verify the InferenceHub platform installation",
	RunE:  runVerify,
}

func init() {
	rootCmd.AddCommand(verifyCmd)
}

func runVerify(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	uiInstance := ui.New(verbose)
	uiInstance.PrintHeader("Verification")

	k8sClient, err := k8s.NewClient(getKubeconfigPath())
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	if err := k8sClient.VerifyConnection(ctx); err != nil {
		return fmt.Errorf("failed to connect to cluster: %w", err)
	}
	uiInstance.Success("Connected to cluster")

	// Check prerequisites
	uiInstance.PrintPhase("Prerequisites Status")
	prereqItems := []ui.StatusItem{
		checkCRD(ctx, k8sClient, "Gateway API CRDs", "gateways.gateway.networking.k8s.io"),
		checkPods(ctx, k8sClient, "cert-manager", "cert-manager", "app.kubernetes.io/instance=cert-manager"),
		checkPods(ctx, k8sClient, "Envoy Gateway", "envoy-gateway-system", ""),
	}
	uiInstance.PrintStatusTable(prereqItems)

	prereqsReady := true
	for _, item := range prereqItems {
		if !item.Success {
			prereqsReady = false
		}
	}
	if !prereqsReady {
		uiInstance.Warn("Some prerequisites are not ready")
		uiInstance.Info("Run: ./scripts/setup-prerequisites.py --cluster-name <name> --domain <domain>")
	}

	// Check platform installation
	uiInstance.PrintPhase("Platform Status")

	pods, err := k8sClient.GetPods(ctx, namespace, "app.kubernetes.io/instance=inferencehub")
	if err != nil || len(pods.Items) == 0 {
		uiInstance.Warn("No InferenceHub pods found in namespace %s", namespace)
		uiInstance.Info("Run 'inferencehub install --config <config.yaml>' to install the platform")
		return nil
	}

	totalCount := 0
	readyCount := 0
	for _, pod := range pods.Items {
		// Skip pods owned by a Job (e.g. migrations)
		isJobPod := false
		for _, owner := range pod.OwnerReferences {
			if owner.Kind == "Job" {
				isJobPod = true
				break
			}
		}
		if isJobPod {
			continue
		}

		totalCount++
		if k8sClient.IsPodReady(&pod) {
			readyCount++
		}
	}

	uiInstance.PrintKeyValue("Total Pods", fmt.Sprintf("%d", totalCount))
	uiInstance.PrintKeyValue("Ready Pods", fmt.Sprintf("%d", readyCount))

	if readyCount == totalCount {
		uiInstance.Success("All pods are ready!")
	} else {
		uiInstance.Warn("Some pods are not ready (%d/%d)", readyCount, totalCount)
		uiInstance.Info("Run 'inferencehub status' for detailed component diagnostics")
	}

	return nil
}

func checkCRD(ctx context.Context, k8sClient *k8s.Client, name, crdName string) ui.StatusItem {
	exists, err := k8sClient.CRDExists(ctx, crdName)
	if err != nil {
		return ui.StatusItem{Name: name, Message: fmt.Sprintf("Error: %v", err), Success: false}
	}
	if !exists {
		return ui.StatusItem{Name: name, Message: "Not installed", Success: false}
	}
	return ui.StatusItem{Name: name, Message: "Installed", Success: true}
}

func checkPods(ctx context.Context, k8sClient *k8s.Client, name, ns, selector string) ui.StatusItem {
	pods, err := k8sClient.GetPods(ctx, ns, selector)
	if err != nil {
		return ui.StatusItem{Name: name, Message: fmt.Sprintf("Error: %v", err), Success: false}
	}
	if len(pods.Items) == 0 {
		return ui.StatusItem{Name: name, Message: "No pods found", Success: false}
	}
	readyCount := 0
	for _, pod := range pods.Items {
		if k8sClient.IsPodReady(&pod) {
			readyCount++
		}
	}
	msg := fmt.Sprintf("%d/%d pods ready", readyCount, len(pods.Items))
	return ui.StatusItem{Name: name, Message: msg, Success: readyCount == len(pods.Items)}
}
