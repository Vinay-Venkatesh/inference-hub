package cli

import (
	"context"
	"fmt"

	"github.com/Vinay-Venkatesh/inferencehub-cli/internal/helm"
	"github.com/Vinay-Venkatesh/inferencehub-cli/internal/ui"
	"github.com/spf13/cobra"
)

var (
	confirmUninstall bool
	purge            bool
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall the InferenceHub platform",
	Long: `Uninstall the InferenceHub platform from the Kubernetes cluster.

This command removes the Helm release and optionally deletes persistent data.

WARNING: Use --purge to also delete PVCs and all stored data.

Examples:
  inferencehub uninstall --confirm
  inferencehub uninstall --confirm --purge`,
	RunE: runUninstall,
}

func init() {
	uninstallCmd.Flags().BoolVar(&confirmUninstall, "confirm", false, "Confirm uninstallation (required)")
	uninstallCmd.Flags().BoolVar(&purge, "purge", false, "Remove all data including PVCs (cannot be undone)")
	_ = uninstallCmd.MarkFlagRequired("confirm")
	rootCmd.AddCommand(uninstallCmd)
}

func runUninstall(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	uiInstance := ui.New(verbose)
	uiInstance.PrintHeader("InferenceHub Uninstall")

	uiInstance.Warn("This will remove the InferenceHub platform from namespace: %s", namespace)
	if purge {
		uiInstance.Warn("PURGE MODE: All persistent data (PVCs) will also be deleted!")
	}

	helmClient, err := helm.NewClient(getKubeconfigPath(), namespace, verbose)
	if err != nil {
		return fmt.Errorf("failed to create Helm client: %w", err)
	}

	exists, err := helmClient.ReleaseExists(ctx, "inferencehub")
	if err != nil {
		return fmt.Errorf("failed to check release: %w", err)
	}
	if !exists {
		uiInstance.Warn("No InferenceHub release found in namespace %s", namespace)
		return nil
	}

	uiInstance.StartSpinner("Uninstalling InferenceHub...")
	if err := helmClient.Uninstall(ctx, "inferencehub"); err != nil {
		uiInstance.StopSpinnerWithMessage(false, "Uninstall failed")
		return fmt.Errorf("helm uninstall failed: %w", err)
	}
	uiInstance.StopSpinnerWithMessage(true, "Helm release removed")

	uiInstance.Success("InferenceHub uninstalled from namespace %s", namespace)
	if !purge {
		uiInstance.Info("PVCs were retained. Use --purge to delete data permanently.")
	}

	return nil
}
