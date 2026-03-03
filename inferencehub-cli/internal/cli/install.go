package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Vinay-Venkatesh/inferencehub-cli/internal/config"
	"github.com/Vinay-Venkatesh/inferencehub-cli/internal/orchestrator"
	"github.com/Vinay-Venkatesh/inferencehub-cli/internal/ui"
	"github.com/spf13/cobra"
)

var (
	installConfigFile    string
	installValuesFile    string
	installChartPath     string
	installCloudProvider string
	installAutoApprove   bool
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install InferenceHub on a Kubernetes cluster",
	Long: `Install the InferenceHub platform using config.yaml.

If your cluster already has Gateway API CRDs, cert-manager, Envoy Gateway, and a
Gateway resource, you can skip straight to install. Otherwise run once per cluster:
  ./scripts/setup-prerequisites.py --cluster-name <name> --domain <domain>

The values file is selected automatically from cloudProvider in config.yaml:
  cloudProvider: aws    →  helm/inferencehub/values-aws.yaml  (gp3 storage, IRSA injected from aws.irsaRoleArn)
  cloudProvider: local  →  helm/inferencehub/values-local.yaml

Examples:
  # AWS EKS with Bedrock (cloudProvider: aws set in config.yaml)
  inferencehub install --config inferencehub.yaml

  # Override cloud provider at runtime
  inferencehub install --config inferencehub.yaml --cloud-provider aws

  # Explicit values file (bypasses auto-selection)
  inferencehub install --config inferencehub.yaml --values ./my-overrides.yaml

Required environment variables:
  LITELLM_MASTER_KEY   LiteLLM API key (must start with sk-)

Optional environment variables (for secrets in config.yaml):
  POSTGRES_PASSWORD    External PostgreSQL password
  REDIS_PASSWORD       External Redis password
  LANGFUSE_PUBLIC_KEY  Langfuse public key (if observability enabled)
  LANGFUSE_SECRET_KEY  Langfuse secret key (if observability enabled)`,
	RunE: runInstall,
}

func init() {
	installCmd.Flags().StringVarP(&installConfigFile, "config", "c", "", "Path to config.yaml (required)")
	installCmd.Flags().StringVar(&installCloudProvider, "cloud-provider", "", "Cloud provider: aws, gcp, azure, local (overrides cloudProvider in config)")
	installCmd.Flags().StringVarP(&installValuesFile, "values", "f", "", "Explicit path to Helm values file (overrides --cloud-provider auto-selection)")
	installCmd.Flags().StringVar(&installChartPath, "chart", "", "Path to Helm chart directory (auto-detected if not provided)")
	installCmd.Flags().BoolVar(&installAutoApprove, "auto-approve", false, "Skip confirmation prompt")

	_ = installCmd.MarkFlagRequired("config")

	rootCmd.AddCommand(installCmd)
}

func runInstall(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	uiInstance := ui.New(verbose)

	// Load config
	uiInstance.Info("Loading configuration from %s...", installConfigFile)
	cfg, err := config.Load(installConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Apply CLI overrides
	if namespace != "" {
		cfg.Namespace = namespace
	}
	if kubeconfigPath != "" {
		cfg.KubeconfigPath = kubeconfigPath
	}

	// Resolve chart path
	cfg.ChartPath, err = resolveChartPath(installChartPath)
	if err != nil {
		return err
	}

	// Resolve cloud provider (CLI flag overrides config)
	if installCloudProvider != "" {
		cfg.CloudProvider = installCloudProvider
	}

	// Resolve values file: explicit --values wins, then cloud provider auto-selection
	valuesFile, err := resolveValuesFile(installValuesFile, cfg.CloudProvider, cfg.ChartPath)
	if err != nil {
		return err
	}

	// Validate
	uiInstance.Info("Validating configuration...")
	validationErr, warnings := config.ValidateAndWarn(cfg)
	if validationErr != nil {
		return fmt.Errorf("configuration invalid: %w", validationErr)
	}
	for _, w := range warnings {
		uiInstance.Warn(w)
	}

	// Confirmation
	if !installAutoApprove {
		printInstallSummary(uiInstance, cfg, valuesFile)
		if !uiInstance.Confirm("Proceed with installation?") {
			uiInstance.Warn("Installation cancelled")
			return nil
		}
	}

	// Create and run orchestrator
	orch, err := orchestrator.NewInstallOrchestrator(cfg, valuesFile, uiInstance)
	if err != nil {
		return fmt.Errorf("failed to initialise orchestrator: %w", err)
	}

	return orch.Execute(ctx, installAutoApprove)
}

// printInstallSummary displays what will be installed before confirmation.
func printInstallSummary(uiInstance *ui.UI, cfg *config.Config, valuesFile string) {
	uiInstance.PrintHeader("Installation Summary")
	uiInstance.PrintKeyValue("Cluster", cfg.ClusterName)
	uiInstance.PrintKeyValue("Namespace", cfg.EffectiveNamespace())
	uiInstance.PrintKeyValue("Domain", cfg.Domain)
	uiInstance.PrintKeyValue("Environment", fmt.Sprintf("%s (issuer: %s)", cfg.Environment, cfg.IssuerType()))
	uiInstance.PrintKeyValue("Gateway", fmt.Sprintf("%s/%s", cfg.Gateway.Namespace, cfg.Gateway.Name))
	if cfg.CloudProvider != "" {
		uiInstance.PrintKeyValue("Cloud Provider", cfg.CloudProvider)
	}
	uiInstance.PrintKeyValue("Models", fmt.Sprintf("%d configured", cfg.Models.TotalCount()))
	if cfg.PostgreSQL.IsExternal() {
		uiInstance.PrintKeyValue("PostgreSQL", "external")
	} else {
		uiInstance.PrintKeyValue("PostgreSQL", "in-cluster")
	}
	owRedis := "in-cluster"
	if cfg.Redis.OpenWebUI.IsExternal() {
		owRedis = "external"
	}
	litellmRedis := "in-cluster"
	if cfg.Redis.LiteLLM.IsExternal() {
		litellmRedis = "external"
	}
	uiInstance.PrintKeyValue("Redis (OpenWebUI)", owRedis)
	uiInstance.PrintKeyValue("Redis (LiteLLM)", litellmRedis)
	if cfg.Observability.Enabled {
		uiInstance.PrintKeyValue("Observability", "Langfuse enabled")
	}
	uiInstance.PrintKeyValue("Helm chart", cfg.ChartPath)
	if valuesFile != "" {
		uiInstance.PrintKeyValue("Values file", valuesFile)
	}
	fmt.Println()
}

// resolveValuesFile determines the Helm values file to use for install/upgrade.
//
// Priority (highest to lowest):
//  1. explicit — the --values flag path, validated to exist
//  2. cloudProvider — maps to helm/inferencehub/values-{provider}.yaml inside the chart dir
//  3. empty — no additional values file (chart defaults apply)
func resolveValuesFile(explicit, cloudProvider, chartPath string) (string, error) {
	if explicit != "" {
		if _, err := os.Stat(explicit); err != nil {
			return "", fmt.Errorf("values file not found: %s", explicit)
		}
		return explicit, nil
	}
	if cloudProvider != "" {
		path := filepath.Join(chartPath, fmt.Sprintf("values-%s.yaml", cloudProvider))
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
		return "", fmt.Errorf("no values file found for cloud provider %q (expected: %s)", cloudProvider, path)
	}
	return "", nil
}

// resolveChartPath returns the absolute path to the Helm chart.
// Searches standard locations relative to cwd and executable if override is empty.
func resolveChartPath(override string) (string, error) {
	if override != "" {
		if _, err := os.Stat(filepath.Join(override, "Chart.yaml")); err != nil {
			return "", fmt.Errorf("chart not found at %s: %w", override, err)
		}
		return override, nil
	}

	cwd, _ := os.Getwd()
	candidates := []string{
		filepath.Join(cwd, "helm", "inferencehub"),
		filepath.Join(cwd, "..", "helm", "inferencehub"),
		filepath.Join(cwd, "..", "..", "helm", "inferencehub"),
	}

	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "helm", "inferencehub"),
			filepath.Join(exeDir, "..", "helm", "inferencehub"),
		)
	}

	for _, path := range candidates {
		if _, err := os.Stat(filepath.Join(path, "Chart.yaml")); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("Helm chart not found. Use --chart to specify the path to helm/inferencehub")
}
