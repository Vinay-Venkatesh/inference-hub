package cli

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	kubeconfigPath string
	namespace      string
	verbose        bool
)

var rootCmd = &cobra.Command{
	Use:   "inferencehub",
	Short: "InferenceHub — your central AI inference platform on Kubernetes",
	Long: color.CyanString(`
┌─────────────────────────────────────────────────────────┐
│   InferenceHub                                          │
│   Kubernetes-native LLM Control Plane                   |
|   Unified chat, routing, and observability              │
│   Powered by OpenWebUI · LiteLLM · Langfuse             │
└─────────────────────────────────────────────────────────┘`) + `

  A Kubernetes-native LLM control plane that unifies chat interfaces and 
  model routing and observability (via Langfuse) across multiple providers.

  Workflow:
    1. Install platform:
         inferencehub install --config inferencehub.yaml

    Skip to step 1 if your cluster already has: Gateway API CRDs, cert-manager,
    Envoy Gateway, and a Gateway resource. Otherwise, run once per cluster first:
         ./scripts/setup-prerequisites.py --cluster-name <name> --domain <domain>

  Cloud provider support (auto-selects values file + injects credentials):
    cloudProvider: aws    →  values-aws.yaml  +  aws.litellmRoleArn
    cloudProvider: local  →  values-local.yaml

  Supported model providers: AWS Bedrock, OpenAI, Ollama, Azure OpenAI
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&kubeconfigPath, "kubeconfig", "", "Path to kubeconfig file (default: ~/.kube/config)")
	rootCmd.PersistentFlags().StringVar(&namespace, "namespace", "inferencehub", "Kubernetes namespace")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
}

func getKubeconfigPath() string {
	if kubeconfigPath != "" {
		return kubeconfigPath
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s/.kube/config", home)
}
