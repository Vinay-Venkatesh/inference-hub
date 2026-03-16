package config

import (
	"strings"
	"testing"
)

// validConfig returns a complete valid Config for testing.
func validConfig() *Config {
	return &Config{
		ClusterName: "test-cluster",
		Domain:      "inference.example.com",
		Environment: "production",
		Gateway: GatewayConfig{
			Name:      "inferencehub-gateway",
			Namespace: "envoy-gateway-system",
		},
		Models: ModelsConfig{
			OpenAI: []OpenAIModel{{Name: "gpt-4o", Model: "gpt-4o"}},
		},
	}
}

// TestValidate_ValidConfig ensures a complete valid config passes with no error.
func TestValidate_ValidConfig(t *testing.T) {
	t.Setenv("LITELLM_MASTER_KEY", "sk-test-key")
	t.Setenv("POSTGRES_PASSWORD", "testpw")
	cfg := validConfig()
	if err := Validate(cfg); err != nil {
		t.Errorf("expected no error for valid config, got: %v", err)
	}
}

// TestValidate_MissingClusterName ensures missing clusterName returns an error.
func TestValidate_MissingClusterName(t *testing.T) {
	t.Setenv("LITELLM_MASTER_KEY", "sk-test-key")
	t.Setenv("POSTGRES_PASSWORD", "testpw")
	cfg := validConfig()
	cfg.ClusterName = ""
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for missing clusterName, got nil")
	}
	if !strings.Contains(err.Error(), "clusterName") {
		t.Errorf("error %q should mention 'clusterName'", err.Error())
	}
}

// TestValidate_MissingDomain ensures missing domain returns an error.
func TestValidate_MissingDomain(t *testing.T) {
	t.Setenv("LITELLM_MASTER_KEY", "sk-test-key")
	t.Setenv("POSTGRES_PASSWORD", "testpw")
	cfg := validConfig()
	cfg.Domain = ""
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for missing domain, got nil")
	}
	if !strings.Contains(err.Error(), "domain") {
		t.Errorf("error %q should mention 'domain'", err.Error())
	}
}

// TestValidate_InvalidDomain ensures an invalid domain returns an appropriate error.
func TestValidate_InvalidDomain(t *testing.T) {
	t.Setenv("LITELLM_MASTER_KEY", "sk-test-key")
	t.Setenv("POSTGRES_PASSWORD", "testpw")
	cfg := validConfig()
	cfg.Domain = "not-a-domain"
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for invalid domain, got nil")
	}
	if !strings.Contains(err.Error(), "not a valid FQDN") {
		t.Errorf("error %q should mention 'not a valid FQDN'", err.Error())
	}
}

// TestValidate_MissingEnvironment ensures missing environment returns an error.
func TestValidate_MissingEnvironment(t *testing.T) {
	t.Setenv("LITELLM_MASTER_KEY", "sk-test-key")
	t.Setenv("POSTGRES_PASSWORD", "testpw")
	cfg := validConfig()
	cfg.Environment = ""
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for missing environment, got nil")
	}
	if !strings.Contains(err.Error(), "environment") {
		t.Errorf("error %q should mention 'environment'", err.Error())
	}
}

// TestValidate_MissingModels ensures that no models returns an error about "at least one model".
func TestValidate_MissingModels(t *testing.T) {
	t.Setenv("LITELLM_MASTER_KEY", "sk-test-key")
	t.Setenv("POSTGRES_PASSWORD", "testpw")
	cfg := validConfig()
	cfg.Models = ModelsConfig{}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for missing models, got nil")
	}
	if !strings.Contains(err.Error(), "at least one model") {
		t.Errorf("error %q should mention 'at least one model'", err.Error())
	}
}

// TestValidate_MissingLiteLLMMasterKey ensures that missing LITELLM_MASTER_KEY
// returns an error mentioning the variable name.
func TestValidate_MissingLiteLLMMasterKey(t *testing.T) {
	t.Setenv("POSTGRES_PASSWORD", "testpw")
	// Explicitly unset LITELLM_MASTER_KEY
	t.Setenv("LITELLM_MASTER_KEY", "")
	cfg := validConfig()
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for missing LITELLM_MASTER_KEY, got nil")
	}
	if !strings.Contains(err.Error(), "LITELLM_MASTER_KEY") {
		t.Errorf("error %q should mention 'LITELLM_MASTER_KEY'", err.Error())
	}
}

// TestValidate_LiteLLMMasterKeyBadPrefix ensures that a master key without "sk-"
// prefix returns an error mentioning "must start with".
func TestValidate_LiteLLMMasterKeyBadPrefix(t *testing.T) {
	t.Setenv("LITELLM_MASTER_KEY", "bad-key")
	t.Setenv("POSTGRES_PASSWORD", "testpw")
	cfg := validConfig()
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for bad LITELLM_MASTER_KEY prefix, got nil")
	}
	if !strings.Contains(err.Error(), "must start with") {
		t.Errorf("error %q should mention 'must start with'", err.Error())
	}
}

// TestValidate_MissingGatewayName ensures that empty gateway.name returns an error.
func TestValidate_MissingGatewayName(t *testing.T) {
	t.Setenv("LITELLM_MASTER_KEY", "sk-test-key")
	t.Setenv("POSTGRES_PASSWORD", "testpw")
	cfg := validConfig()
	cfg.Gateway.Name = ""
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for missing gateway.name, got nil")
	}
	if !strings.Contains(err.Error(), "gateway.name") {
		t.Errorf("error %q should mention 'gateway.name'", err.Error())
	}
}

// TestValidate_BedrockModelMissingRegion ensures that a Bedrock model without
// a region returns an error mentioning "region".
func TestValidate_BedrockModelMissingRegion(t *testing.T) {
	t.Setenv("LITELLM_MASTER_KEY", "sk-test-key")
	t.Setenv("POSTGRES_PASSWORD", "testpw")
	cfg := validConfig()
	cfg.Models = ModelsConfig{
		Bedrock: []BedrockModel{
			{Name: "claude-3", Model: "anthropic.claude-3-sonnet-20240229-v1:0", Region: ""},
		},
	}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for Bedrock model missing region, got nil")
	}
	if !strings.Contains(err.Error(), "region") {
		t.Errorf("error %q should mention 'region'", err.Error())
	}
}

// TestValidate_OllamaModelMissingAPIBase ensures that an Ollama model without
// apiBase returns an error mentioning "apiBase".
func TestValidate_OllamaModelMissingAPIBase(t *testing.T) {
	t.Setenv("LITELLM_MASTER_KEY", "sk-test-key")
	t.Setenv("POSTGRES_PASSWORD", "testpw")
	cfg := validConfig()
	cfg.Models = ModelsConfig{
		Ollama: []OllamaModel{
			{Name: "llama3", Model: "llama3", APIBase: ""},
		},
	}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for Ollama model missing apiBase, got nil")
	}
	if !strings.Contains(err.Error(), "apiBase") {
		t.Errorf("error %q should mention 'apiBase'", err.Error())
	}
}

// TestValidateAndWarn_StagingEnvWarning ensures that a staging environment
// produces a warning about TLS certificates.
func TestValidateAndWarn_StagingEnvWarning(t *testing.T) {
	t.Setenv("LITELLM_MASTER_KEY", "sk-test-key")
	t.Setenv("POSTGRES_PASSWORD", "testpw")
	cfg := validConfig()
	cfg.Environment = "staging"
	err, warnings := ValidateAndWarn(cfg)
	if err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
	found := false
	for _, w := range warnings {
		if strings.Contains(strings.ToLower(w), "tls") || strings.Contains(strings.ToLower(w), "certificate") || strings.Contains(w, "staging") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected a TLS/staging warning, got warnings: %v", warnings)
	}
}

// TestValidateAndWarn_BedrockWithoutIRSA ensures that Bedrock models without
// aws.litellmRoleArn produce a warning.
func TestValidateAndWarn_BedrockWithoutIRSA(t *testing.T) {
	t.Setenv("LITELLM_MASTER_KEY", "sk-test-key")
	t.Setenv("POSTGRES_PASSWORD", "testpw")
	cfg := validConfig()
	cfg.Environment = "production"
	cfg.Models = ModelsConfig{
		Bedrock: []BedrockModel{
			{Name: "claude-3", Model: "anthropic.claude-3-sonnet-20240229-v1:0", Region: "us-east-1"},
		},
	}
	cfg.AWS = AWSConfig{LiteLLMRoleARN: ""}
	err, warnings := ValidateAndWarn(cfg)
	if err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "Bedrock") || strings.Contains(w, "litellmRoleArn") || strings.Contains(w, "IRSA") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected a Bedrock/IRSA warning, got warnings: %v", warnings)
	}
}

// TestValidateAndWarn_InClusterPostgresWarning ensures that in-cluster postgres
// produces a production warning.
func TestValidateAndWarn_InClusterPostgresWarning(t *testing.T) {
	t.Setenv("LITELLM_MASTER_KEY", "sk-test-key")
	t.Setenv("POSTGRES_PASSWORD", "testpw")
	cfg := validConfig()
	cfg.Environment = "production"
	// No external PostgreSQL configured — in-cluster by default
	err, warnings := ValidateAndWarn(cfg)
	if err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
	found := false
	for _, w := range warnings {
		if strings.Contains(strings.ToLower(w), "postgres") || strings.Contains(strings.ToLower(w), "database") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected an in-cluster PostgreSQL warning, got warnings: %v", warnings)
	}
}

// TestValidateAndWarn_ProtectedPassthroughKeyWarning ensures that setting
// openwebui.openaiBaseApiUrl in passthrough produces a warning.
func TestValidateAndWarn_ProtectedPassthroughKeyWarning(t *testing.T) {
	t.Setenv("LITELLM_MASTER_KEY", "sk-test-key")
	t.Setenv("POSTGRES_PASSWORD", "testpw")
	cfg := validConfig()
	cfg.Environment = "production"
	cfg.OpenWebUI = map[string]interface{}{
		"openaiBaseApiUrl": "http://custom-litellm:4000/v1",
	}
	err, warnings := ValidateAndWarn(cfg)
	if err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "openaiBaseApiUrl") || strings.Contains(w, "managed by InferenceHub") || strings.Contains(w, "overridden") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected a protected passthrough key warning for openaiBaseApiUrl, got warnings: %v", warnings)
	}
}
