package orchestrator

import (
	"context"
	"fmt"
	"time"

	"github.com/Vinay-Venkatesh/inferencehub-cli/internal/config"
	"github.com/Vinay-Venkatesh/inferencehub-cli/internal/helm"
	"github.com/Vinay-Venkatesh/inferencehub-cli/internal/k8s"
	"github.com/Vinay-Venkatesh/inferencehub-cli/internal/ui"
	"github.com/Vinay-Venkatesh/inferencehub-cli/internal/verify"
)

// InstallOrchestrator orchestrates the InferenceHub installation.
type InstallOrchestrator struct {
	config     *config.Config
	valuesFile string // optional provider-specific values file (e.g. values-aws.yaml)
	k8sClient  *k8s.Client
	helmClient *helm.Client
	verifier   *verify.Orchestrator
	ui         *ui.UI
}

// NewInstallOrchestrator creates a new installation orchestrator.
// valuesFile is an optional path to a provider-specific Helm values file
// (e.g. helm/inferencehub/values-aws.yaml). Pass empty string to omit.
func NewInstallOrchestrator(cfg *config.Config, valuesFile string, uiInstance *ui.UI) (*InstallOrchestrator, error) {
	k8sClient, err := k8s.NewClient(cfg.KubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	helmClient, err := helm.NewClient(cfg.KubeconfigPath, cfg.EffectiveNamespace(), uiInstance != nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Helm client: %w", err)
	}

	return &InstallOrchestrator{
		config:     cfg,
		valuesFile: valuesFile,
		k8sClient:  k8sClient,
		helmClient: helmClient,
		verifier:   verify.NewOrchestrator(k8sClient, uiInstance, cfg),
		ui:         uiInstance,
	}, nil
}

// Execute runs the full installation flow.
//
// Phase 1: Validate — cluster connectivity + gateway existence check
// Phase 2: Install  — namespace creation + Helm install
// Phase 3: Wait     — pod readiness
// Phase 4: Verify   — service/route/cert checks
func (o *InstallOrchestrator) Execute(ctx context.Context, autoApprove bool) error {
	o.ui.PrintHeader("InferenceHub Installation")

	if err := o.phaseValidate(ctx); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if err := o.phaseInstall(ctx); err != nil {
		return fmt.Errorf("installation failed: %w", err)
	}

	if err := o.phaseWait(ctx); err != nil {
		o.ui.Warn("Some pods are not ready yet - they may still be starting up")
		o.ui.Info("Run 'kubectl get pods -n %s' to check status", o.config.EffectiveNamespace())
	}

	if err := o.phaseVerify(ctx); err != nil {
		o.ui.Warn("Verification completed with errors (platform may still be initialising)")
		o.ui.Info("Run 'inferencehub verify' later to check status")
	}

	o.printSummary()
	return nil
}

// phaseValidate verifies cluster access and pre-flight conditions.
func (o *InstallOrchestrator) phaseValidate(ctx context.Context) error {
	o.ui.PrintPhase("Phase 1: Validate")

	o.ui.StartSpinner("Connecting to Kubernetes cluster...")
	if err := o.k8sClient.VerifyConnection(ctx); err != nil {
		o.ui.StopSpinnerWithMessage(false, "Cannot connect to cluster")
		return err
	}
	version, _ := o.k8sClient.GetServerVersion(ctx)
	o.ui.StopSpinnerWithMessage(true, fmt.Sprintf("Connected (Kubernetes %s)", version))

	// Verify the gateway exists — fail fast if prerequisites weren't run
	o.ui.StartSpinner(fmt.Sprintf("Checking Gateway '%s/%s'...", o.config.Gateway.Namespace, o.config.Gateway.Name))
	exists, err := o.k8sClient.CustomResourceExists(ctx, k8s.GatewayGVR(), o.config.Gateway.Namespace, o.config.Gateway.Name)
	if err != nil || !exists {
		o.ui.StopSpinnerWithMessage(false, "Gateway not found")
		return fmt.Errorf(
			"Gateway '%s' not found in namespace '%s'.\n"+
				"  Run: ./scripts/setup-prerequisites.py --cluster-name %s --domain %s",
			o.config.Gateway.Name, o.config.Gateway.Namespace,
			o.config.ClusterName, o.config.Domain,
		)
	}
	o.ui.StopSpinnerWithMessage(true, "Gateway exists")

	return nil
}

// phaseInstall creates the namespace and runs helm install.
func (o *InstallOrchestrator) phaseInstall(ctx context.Context) error {
	o.ui.PrintPhase("Phase 2: Install")

	// Abort if release already exists
	exists, err := o.helmClient.ReleaseExists(ctx, "inferencehub")
	if err != nil {
		return err
	}
	if exists {
		o.ui.Warn("Release 'inferencehub' already exists")
		o.ui.Info("Use 'inferencehub upgrade --config config.yaml' to apply changes")
		return fmt.Errorf("release already exists")
	}

	// Load provider-specific values file (e.g. values-aws.yaml)
	fileValues, err := helm.LoadValuesFile(o.valuesFile)
	if err != nil {
		return fmt.Errorf("failed to load values file: %w", err)
	}
	if o.valuesFile != "" {
		o.ui.Info("Loaded values file: %s", o.valuesFile)
	}

	// Generate programmatic overrides (these win over file values)
	o.ui.Info("Generating Helm values...")
	overrides, err := helm.GenerateOverrides(o.config, helm.DefaultReleaseName, ctx, o.k8sClient)
	if err != nil {
		return fmt.Errorf("failed to generate Helm values: %w", err)
	}

	// Merge: file values as base, programmatic overrides take precedence
	merged := helm.MergeValues(fileValues, overrides)

	// Storage class feedback (navigate the nested map)
	if pg, ok := overrides["postgresql"].(map[string]interface{}); ok {
		if persistence, ok := pg["persistence"].(map[string]interface{}); ok {
			if sc, ok := persistence["storageClass"].(string); ok && sc != "" {
				o.ui.Info("Auto-detected storage class: %s", sc)
			}
		}
	}

	// Create namespace
	o.ui.Info("Creating namespace '%s'...", o.config.EffectiveNamespace())
	if err := o.k8sClient.CreateNamespace(ctx, o.config.EffectiveNamespace()); err != nil {
		return err
	}

	// Debug: confirm networking values are in the merged map
	if net, ok := merged["networking"].(map[string]interface{}); ok {
		if gw, ok := net["gatewayAPI"].(map[string]interface{}); ok {
			o.ui.Info("Helm networking.gatewayAPI.hostname = %v", gw["hostname"])
		}
	} else {
		o.ui.Warn("networking key missing from merged values — domain will not be set")
	}

	// Helm install
	o.ui.StartSpinner("Installing InferenceHub...")
	if _, err := o.helmClient.Install(ctx, "inferencehub", o.config.ChartPath, merged); err != nil {
		o.ui.StopSpinnerWithMessage(false, "Helm install failed")
		return err
	}
	o.ui.StopSpinnerWithMessage(true, "Helm install complete")

	return nil
}

// phaseWait waits for all component pods to become ready.
func (o *InstallOrchestrator) phaseWait(ctx context.Context) error {
	o.ui.PrintPhase("Phase 3: Wait for Pods")

	components := map[string]string{
		"openwebui":  "app.kubernetes.io/component=ui",
		"litellm":    "app.kubernetes.io/component=gateway",
		"postgresql": "app.kubernetes.io/component=database",
		"redis":      "app.kubernetes.io/component=cache",
	}

	var lastErr error
	for name, selector := range components {
		o.ui.StartSpinner(fmt.Sprintf("Waiting for %s...", name))
		err := o.k8sClient.WaitForPodsReady(ctx, o.config.EffectiveNamespace(), selector, 5*time.Minute)
		if err != nil {
			o.ui.StopSpinnerWithMessage(false, fmt.Sprintf("%s not ready yet", name))
			lastErr = err
		} else {
			o.ui.StopSpinnerWithMessage(true, fmt.Sprintf("%s ready", name))
		}
	}

	return lastErr
}

// phaseVerify runs the end-to-end verification checks.
func (o *InstallOrchestrator) phaseVerify(ctx context.Context) error {
	_, err := o.verifier.VerifyAll(ctx)
	return err
}

// printSummary displays post-installation information.
func (o *InstallOrchestrator) printSummary() {
	o.ui.PrintPhase("Installation Complete")

	scheme := "http"
	if o.config.IssuerType() != "" {
		scheme = "https"
	}

	o.ui.Success("InferenceHub installed successfully")
	o.ui.PrintKeyValue("URL", fmt.Sprintf("%s://%s", scheme, o.config.Domain))
	o.ui.PrintKeyValue("Namespace", o.config.EffectiveNamespace())
	o.ui.PrintKeyValue("Models", fmt.Sprintf("%d configured", o.config.Models.TotalCount()))
	o.ui.Println("")
	o.ui.Info("Next steps:")
	o.ui.Info("  1. Point DNS: %s → $(kubectl get gateway %s -n %s -o jsonpath='{.status.addresses[0].value}')",
		o.config.Domain, o.config.Gateway.Name, o.config.Gateway.Namespace)
	o.ui.Info("  2. Wait for TLS certificate (if TLS enabled): kubectl get certificate -n %s", o.config.EffectiveNamespace())
	o.ui.Info("  3. Open %s://%s", scheme, o.config.Domain)
	o.ui.Println("")
	o.ui.Info("Useful commands:")
	o.ui.Info("  inferencehub status    - Check pod and service status")
	o.ui.Info("  inferencehub verify    - Run end-to-end verification")
	o.ui.Info("  inferencehub upgrade --config config.yaml  - Apply config changes")
}
