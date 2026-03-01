package verify

import (
	"context"
	"fmt"
	"time"

	"github.com/Vinay-Venkatesh/inferencehub-cli/internal/config"
	"github.com/Vinay-Venkatesh/inferencehub-cli/internal/k8s"
	"github.com/Vinay-Venkatesh/inferencehub-cli/internal/ui"
)

// Orchestrator runs all verification checks.
type Orchestrator struct {
	k8sClient *k8s.Client
	ui        *ui.UI
	config    *config.Config
}

// NewOrchestrator creates a verification orchestrator.
func NewOrchestrator(k8sClient *k8s.Client, uiInstance *ui.UI, cfg *config.Config) *Orchestrator {
	return &Orchestrator{
		k8sClient: k8sClient,
		ui:        uiInstance,
		config:    cfg,
	}
}

// VerifyAll runs all verification steps and returns a summary.
func (o *Orchestrator) VerifyAll(ctx context.Context) (*VerificationSummary, error) {
	o.ui.PrintPhase("Phase 4: Verify")

	summary := &VerificationSummary{}
	summary.Add(o.verifyPods(ctx))
	summary.Add(o.verifyServices(ctx))
	summary.Add(o.verifyHTTPRoute(ctx))

	if o.config.IssuerType() == "letsencrypt-prod" {
		summary.Add(o.verifyCertificate(ctx))
	}

	o.displayResults(summary)

	if !summary.AllPassed() {
		return summary, fmt.Errorf("verification: %d/%d checks passed", summary.SuccessCount, summary.TotalSteps)
	}
	return summary, nil
}

// verifyPods checks that all component pods are running and ready.
func (o *Orchestrator) verifyPods(ctx context.Context) VerificationResult {
	start := time.Now()
	result := VerificationResult{Step: "Pod Health"}

	componentSelectors := map[string]string{
		"openwebui":  "app.kubernetes.io/component=ui",
		"litellm":    "app.kubernetes.io/component=gateway",
		"postgresql": "app.kubernetes.io/component=database",
		"redis":      "app.kubernetes.io/component=cache",
	}

	for name, selector := range componentSelectors {
		pods, err := o.k8sClient.GetPods(ctx, o.config.EffectiveNamespace(), selector)
		if err != nil {
			result.Message = fmt.Sprintf("cannot get %s pods: %v", name, err)
			result.Duration = time.Since(start)
			return result
		}
		if len(pods.Items) == 0 {
			result.Message = fmt.Sprintf("no %s pods found", name)
			result.Duration = time.Since(start)
			return result
		}
		for _, pod := range pods.Items {
			if !o.k8sClient.IsPodReady(&pod) {
				result.Message = fmt.Sprintf("pod %s is not ready", pod.Name)
				result.Duration = time.Since(start)
				return result
			}
		}
	}

	result.Success = true
	result.Message = "all pods running"
	result.Duration = time.Since(start)
	return result
}

// verifyServices checks that all expected services exist.
func (o *Orchestrator) verifyServices(ctx context.Context) VerificationResult {
	start := time.Now()
	result := VerificationResult{Step: "Services"}

	// Release name defaults to "inferencehub" when installed with default release name
	releaseName := "inferencehub"
	services := []string{
		fmt.Sprintf("%s-openwebui", releaseName),
		fmt.Sprintf("%s-litellm", releaseName),
		fmt.Sprintf("%s-postgresql", releaseName),
		fmt.Sprintf("%s-redis", releaseName),
	}

	for _, svc := range services {
		exists, err := o.k8sClient.ServiceExists(ctx, o.config.EffectiveNamespace(), svc)
		if err != nil {
			result.Message = fmt.Sprintf("cannot check service %s: %v", svc, err)
			result.Duration = time.Since(start)
			return result
		}
		if !exists {
			result.Message = fmt.Sprintf("service %s not found", svc)
			result.Duration = time.Since(start)
			return result
		}
	}

	result.Success = true
	result.Message = "all services present"
	result.Duration = time.Since(start)
	return result
}

// verifyHTTPRoute checks that the InferenceHub HTTPRoute exists.
func (o *Orchestrator) verifyHTTPRoute(ctx context.Context) VerificationResult {
	start := time.Now()
	result := VerificationResult{Step: "HTTPRoute"}

	exists, err := o.k8sClient.CustomResourceExists(
		ctx, k8s.HTTPRouteGVR(), o.config.EffectiveNamespace(), "inferencehub-httproute",
	)
	if err != nil {
		result.Message = fmt.Sprintf("cannot check HTTPRoute: %v", err)
		result.Duration = time.Since(start)
		return result
	}
	if !exists {
		result.Message = "HTTPRoute not found"
		result.Duration = time.Since(start)
		return result
	}

	result.Success = true
	result.Message = "HTTPRoute exists"
	result.Duration = time.Since(start)
	return result
}

// verifyCertificate checks that the TLS certificate exists (production only).
func (o *Orchestrator) verifyCertificate(ctx context.Context) VerificationResult {
	start := time.Now()
	result := VerificationResult{Step: "TLS Certificate"}

	exists, err := o.k8sClient.CustomResourceExists(
		ctx, k8s.CertificateGVR(), o.config.EffectiveNamespace(), "inferencehub-tls",
	)
	if err != nil {
		result.Message = fmt.Sprintf("cannot check Certificate: %v", err)
		result.Duration = time.Since(start)
		return result
	}
	if !exists {
		result.Message = "Certificate not found"
		result.Duration = time.Since(start)
		return result
	}

	result.Success = true
	result.Message = "Certificate exists"
	result.Duration = time.Since(start)
	return result
}

// displayResults prints verification results as a status table.
func (o *Orchestrator) displayResults(summary *VerificationSummary) {
	o.ui.Println("")
	o.ui.PrintSeparator()

	items := make([]ui.StatusItem, len(summary.Results))
	for i, r := range summary.Results {
		items[i] = ui.StatusItem{
			Name:    r.Step,
			Message: r.Message,
			Details: fmt.Sprintf("%v", r.Duration.Round(time.Millisecond)),
			Success: r.Success,
		}
	}
	o.ui.PrintStatusTable(items)

	o.ui.PrintSeparator()
	o.ui.Printf("\n%d/%d checks passed (%.2fs)\n",
		summary.SuccessCount, summary.TotalSteps, summary.TotalTime.Seconds())
}
