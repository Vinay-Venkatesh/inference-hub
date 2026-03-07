package cli

import (
	"fmt"
	"os"

	"github.com/Vinay-Venkatesh/inferencehub-cli/internal/config"
	"github.com/Vinay-Venkatesh/inferencehub-cli/internal/ui"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage InferenceHub configuration",
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate a starter config.yaml",
	RunE:  runConfigInit,
}

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate a config.yaml file",
	RunE:  runConfigValidate,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show the effective configuration after env var interpolation",
	RunE:  runConfigShow,
}

var (
	configOutputPath string
	configInputPath  string
)

func init() {
	configInitCmd.Flags().StringVarP(&configOutputPath, "output", "o", "inferencehub.yaml", "Output file path")

	configValidateCmd.Flags().StringVarP(&configInputPath, "config", "c", "", "Path to config.yaml (required)")
	_ = configValidateCmd.MarkFlagRequired("config")

	configShowCmd.Flags().StringVarP(&configInputPath, "config", "c", "", "Path to config.yaml (required)")
	_ = configShowCmd.MarkFlagRequired("config")

	configCmd.AddCommand(configInitCmd, configValidateCmd, configShowCmd)
	rootCmd.AddCommand(configCmd)
}

func runConfigInit(_ *cobra.Command, _ []string) error {
	uiInstance := ui.New(verbose)

	if _, err := os.Stat(configOutputPath); err == nil {
		if !uiInstance.Confirm(fmt.Sprintf("%s already exists. Overwrite?", configOutputPath)) {
			return nil
		}
	}

	cfg := starterConfig()
	if err := config.SaveToFile(cfg, configOutputPath); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	uiInstance.Success("Created %s", configOutputPath)
	uiInstance.Info("Edit the file to set your values, then:")
	uiInstance.Info("  1. Set environment variables (see .env.example)")
	uiInstance.Info("  2. If cluster lacks cert-manager/Envoy Gateway/Gateway API CRDs:")
	uiInstance.Info("       ./scripts/setup-prerequisites.py --cluster-name <name> --domain <domain>")
	uiInstance.Info("  3. Install: inferencehub install --config %s", configOutputPath)

	return nil
}

func runConfigValidate(_ *cobra.Command, _ []string) error {
	uiInstance := ui.New(verbose)

	cfg, err := config.Load(configInputPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := config.Validate(cfg); err != nil {
		uiInstance.Error("Validation failed: %v", err)
		return err
	}

	uiInstance.Success("Configuration is valid")
	uiInstance.PrintKeyValue("Cluster", cfg.ClusterName)
	uiInstance.PrintKeyValue("Domain", cfg.Domain)
	uiInstance.PrintKeyValue("Environment", fmt.Sprintf("%s (issuer: %s)", cfg.Environment, cfg.IssuerType()))
	if cfg.CloudProvider != "" {
		uiInstance.PrintKeyValue("Cloud Provider", fmt.Sprintf("%s (values-file: values-%s.yaml)", cfg.CloudProvider, cfg.CloudProvider))
	}
	uiInstance.PrintKeyValue("Models", fmt.Sprintf("%d configured", cfg.Models.TotalCount()))

	return nil
}

func runConfigShow(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load(configInputPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Show which .env files were loaded
	envFiles := config.GetLoadedEnvFiles()
	if len(envFiles) > 0 {
		fmt.Println("# Loaded .env files:")
		for _, f := range envFiles {
			fmt.Printf("#   %s\n", f)
		}
		fmt.Println()
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	fmt.Print(string(data))

	return nil
}

// starterConfig returns a well-commented template config for users.
func starterConfig() *config.Config {
	return &config.Config{
		ClusterName:   "my-cluster",
		Domain:        "inferencehub.ai",
		Environment:   "staging",
		Namespace:     "inferencehub",
		CloudProvider: "aws",
		Gateway: config.GatewayConfig{
			Name:      "inferencehub-gateway",
			Namespace: "envoy-gateway-system",
		},
		Models: config.ModelsConfig{
			Bedrock: []config.BedrockModel{
				{
					Name:   "claude-sonnet",
					Model:  "anthropic.claude-3-5-sonnet-20241022-v2:0",
					Region: "us-east-1",
				},
			},
		},
		Observability: config.ObservabilityConfig{
			Enabled: false,
			Langfuse: config.LangfuseConfig{
				Host:      "https://cloud.langfuse.com",
				PublicKey: "${LANGFUSE_PUBLIC_KEY}",
				SecretKey: "${LANGFUSE_SECRET_KEY}",
			},
		},
		AWS: config.AWSConfig{
			LiteLLMRoleARN: "",
		},
	}
}

func hasEnvVars() bool {
	for _, v := range []string{"LITELLM_MASTER_KEY", "POSTGRES_PASSWORD", "REDIS_PASSWORD", "LANGFUSE_PUBLIC_KEY"} {
		if os.Getenv(v) != "" {
			return true
		}
	}
	return false
}
