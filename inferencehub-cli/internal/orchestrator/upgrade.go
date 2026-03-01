package orchestrator

import (
	"context"
	"fmt"

	"github.com/Vinay-Venkatesh/inferencehub-cli/internal/config"
	"github.com/Vinay-Venkatesh/inferencehub-cli/internal/helm"
	"github.com/Vinay-Venkatesh/inferencehub-cli/internal/k8s"
	"github.com/Vinay-Venkatesh/inferencehub-cli/internal/ui"
	"github.com/Vinay-Venkatesh/inferencehub-cli/internal/verify"
)

// UpgradeOrchestrator orchestrates the InferenceHub upgrade.
type UpgradeOrchestrator struct {
	config     *config.Config
	valuesFile string // optional provider-specific values file (e.g. values-aws.yaml)
	k8sClient  *k8s.Client
	helmClient *helm.Client
	verifier   *verify.Orchestrator
	ui         *ui.UI
}

// NewUpgradeOrchestrator creates a new upgrade orchestrator.
// valuesFile is an optional path to a provider-specific Helm values file. Pass empty string to omit.
func NewUpgradeOrchestrator(cfg *config.Config, valuesFile string, uiInstance *ui.UI) (*UpgradeOrchestrator, error) {
	k8sClient, err := k8s.NewClient(cfg.KubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	helmClient, err := helm.NewClient(cfg.KubeconfigPath, cfg.EffectiveNamespace(), uiInstance != nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Helm client: %w", err)
	}

	return &UpgradeOrchestrator{
		config:     cfg,
		valuesFile: valuesFile,
		k8sClient:  k8sClient,
		helmClient: helmClient,
		verifier:   verify.NewOrchestrator(k8sClient, uiInstance, cfg),
		ui:         uiInstance,
	}, nil
}

// Execute runs the upgrade flow:
// Phase 1: Verify existing release
// Phase 2: Helm upgrade
// Phase 3: Verification
func (o *UpgradeOrchestrator) Execute(ctx context.Context) error {
	o.ui.PrintHeader("InferenceHub Upgrade")

	if err := o.phaseVerifyExisting(ctx); err != nil {
		return fmt.Errorf("pre-upgrade check failed: %w", err)
	}

	if err := o.phaseUpgrade(ctx); err != nil {
		return fmt.Errorf("upgrade failed: %w", err)
	}

	if err := o.phaseVerify(ctx); err != nil {
		o.ui.Warn("Verification completed with errors (platform may still be restarting)")
		o.ui.Info("Run 'inferencehub verify' later to check status")
	}

	o.printSummary()
	return nil
}

// phaseVerifyExisting checks that a release already exists.
func (o *UpgradeOrchestrator) phaseVerifyExisting(ctx context.Context) error {
	o.ui.PrintPhase("Phase 1: Check Existing Installation")

	o.ui.StartSpinner("Looking for existing release...")
	exists, err := o.helmClient.ReleaseExists(ctx, "inferencehub")
	if err != nil {
		o.ui.StopSpinnerWithMessage(false, "Cannot check release")
		return err
	}
	if !exists {
		o.ui.StopSpinnerWithMessage(false, "Release not found")
		return fmt.Errorf("no InferenceHub release found in namespace '%s'. Run 'inferencehub install' first", o.config.EffectiveNamespace())
	}

	release, err := o.helmClient.Get(ctx, "inferencehub")
	if err != nil {
		o.ui.StopSpinnerWithMessage(false, "Cannot get release info")
		return err
	}

	o.ui.StopSpinnerWithMessage(true, fmt.Sprintf("Found release (status: %s)", release.Info.Status))
	o.ui.PrintKeyValue("Chart", release.Chart.Name())
	o.ui.PrintKeyValue("App Version", release.Chart.AppVersion())

	return nil
}

// phaseUpgrade generates overrides and runs helm upgrade.
func (o *UpgradeOrchestrator) phaseUpgrade(ctx context.Context) error {
	o.ui.PrintPhase("Phase 2: Upgrade")

	// Load provider-specific values file (e.g. values-aws.yaml)
	fileValues, err := helm.LoadValuesFile(o.valuesFile)
	if err != nil {
		return fmt.Errorf("failed to load values file: %w", err)
	}
	if o.valuesFile != "" {
		o.ui.Info("Loaded values file: %s", o.valuesFile)
	}

	o.ui.Info("Generating Helm values...")
	overrides, err := helm.GenerateOverrides(o.config, ctx, o.k8sClient)
	if err != nil {
		return fmt.Errorf("failed to generate Helm values: %w", err)
	}

	// Merge: file values as base, programmatic overrides take precedence
	merged := helm.MergeValues(fileValues, overrides)

	if sc, ok := overrides["postgresql.persistence.storageClass"].(string); ok && sc != "" {
		o.ui.Info("Storage class: %s", sc)
	}

	o.ui.StartSpinner("Upgrading InferenceHub via Helm...")
	if _, err := o.helmClient.Upgrade(ctx, "inferencehub", o.config.ChartPath, merged); err != nil {
		o.ui.StopSpinnerWithMessage(false, "Helm upgrade failed")
		return err
	}
	o.ui.StopSpinnerWithMessage(true, "Helm upgrade complete")

	return nil
}

// phaseVerify runs the end-to-end verification checks.
func (o *UpgradeOrchestrator) phaseVerify(ctx context.Context) error {
	_, err := o.verifier.VerifyAll(ctx)
	return err
}

// printSummary displays post-upgrade information.
func (o *UpgradeOrchestrator) printSummary() {
	o.ui.PrintPhase("Upgrade Complete")

	scheme := "https"
	if o.config.IssuerType() == "letsencrypt-staging" {
		scheme = "https"
	}

	o.ui.Success("InferenceHub upgraded successfully")
	o.ui.PrintKeyValue("URL", fmt.Sprintf("%s://%s", scheme, o.config.Domain))
	o.ui.PrintKeyValue("Namespace", o.config.EffectiveNamespace())
	o.ui.Println("")
	o.ui.Info("Useful commands:")
	o.ui.Info("  inferencehub status   - Check status")
	o.ui.Info("  inferencehub verify   - Run verification")
}
