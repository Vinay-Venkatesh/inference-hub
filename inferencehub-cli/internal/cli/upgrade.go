package cli

import (
	"context"
	"fmt"

	"github.com/Vinay-Venkatesh/inferencehub-cli/internal/config"
	"github.com/Vinay-Venkatesh/inferencehub-cli/internal/orchestrator"
	"github.com/Vinay-Venkatesh/inferencehub-cli/internal/ui"
	"github.com/spf13/cobra"
)

var (
	upgradeConfigFile    string
	upgradeValuesFile    string
	upgradeCloudProvider string
	upgradeAutoApprove   bool
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade an existing InferenceHub installation",
	Long: `Apply configuration changes or upgrade component versions.

Examples:
  inferencehub upgrade --config config.yaml
  inferencehub upgrade --config config.yaml --values helm/inferencehub/values-aws.yaml`,
	RunE: runUpgrade,
}

func init() {
	upgradeCmd.Flags().StringVarP(&upgradeConfigFile, "config", "c", "", "Path to config.yaml (required)")
	upgradeCmd.Flags().StringVar(&upgradeCloudProvider, "cloud-provider", "", "Cloud provider: aws, gcp, azure, local (overrides cloudProvider in config)")
	upgradeCmd.Flags().StringVarP(&upgradeValuesFile, "values", "f", "", "Explicit path to Helm values file (overrides --cloud-provider auto-selection)")
	upgradeCmd.Flags().BoolVar(&upgradeAutoApprove, "auto-approve", false, "Skip confirmation prompt")

	_ = upgradeCmd.MarkFlagRequired("config")

	rootCmd.AddCommand(upgradeCmd)
}

func runUpgrade(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	uiInstance := ui.New(verbose)

	uiInstance.Info("Loading configuration from %s...", upgradeConfigFile)
	cfg, err := config.Load(upgradeConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if namespace != "" {
		cfg.Namespace = namespace
	}
	if kubeconfigPath != "" {
		cfg.KubeconfigPath = kubeconfigPath
	}

	cfg.ChartPath, err = resolveChartPath("")
	if err != nil {
		return err
	}

	// Resolve cloud provider (CLI flag overrides config)
	if upgradeCloudProvider != "" {
		cfg.CloudProvider = upgradeCloudProvider
	}

	// Resolve values file: explicit --values wins, then cloud provider auto-selection
	valuesFile, err := resolveValuesFile(upgradeValuesFile, cfg.CloudProvider, cfg.ChartPath)
	if err != nil {
		return err
	}

	if err := config.Validate(cfg); err != nil {
		return fmt.Errorf("configuration invalid: %w", err)
	}

	orch, err := orchestrator.NewUpgradeOrchestrator(cfg, valuesFile, uiInstance)
	if err != nil {
		return fmt.Errorf("failed to initialise orchestrator: %w", err)
	}

	return orch.Execute(ctx)
}
