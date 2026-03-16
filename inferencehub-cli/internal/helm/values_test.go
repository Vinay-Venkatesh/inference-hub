package helm

import (
	"testing"

	"github.com/Vinay-Venkatesh/inferencehub-cli/internal/config"
)

// minimalConfig returns a minimal valid config for testing.
func minimalConfig() *config.Config {
	return &config.Config{
		ClusterName: "test-cluster",
		Domain:      "test.example.com",
		Environment: "staging",
		Gateway:     config.GatewayConfig{Name: "inferencehub-gateway", Namespace: "envoy-gateway-system"},
		Models: config.ModelsConfig{
			OpenAI: []config.OpenAIModel{{Name: "gpt-4o", Model: "gpt-4o"}},
		},
	}
}

// getNestedVal walks path in a nested map[string]interface{} and returns the
// value at the leaf, or nil if any key is missing or a non-map is encountered mid-path.
func getNestedVal(m map[string]interface{}, path ...string) interface{} {
	cur := m
	for i, key := range path {
		val, ok := cur[key]
		if !ok {
			return nil
		}
		if i == len(path)-1 {
			return val
		}
		next, ok := val.(map[string]interface{})
		if !ok {
			return nil
		}
		cur = next
	}
	return nil
}

// getEnvVar finds an env var entry by name from openwebui.extraEnvVars.
func getEnvVar(t *testing.T, overrides map[string]interface{}, name string) map[string]interface{} {
	t.Helper()
	owRaw, ok := overrides["openwebui"]
	if !ok {
		t.Fatalf("overrides missing 'openwebui' key")
	}
	owMap, ok := owRaw.(map[string]interface{})
	if !ok {
		t.Fatalf("overrides['openwebui'] is not a map")
	}
	extraRaw, ok := owMap["extraEnvVars"]
	if !ok {
		t.Fatalf("openwebui missing 'extraEnvVars' key")
	}
	switch v := extraRaw.(type) {
	case []map[string]interface{}:
		for _, entry := range v {
			if n, _ := entry["name"].(string); n == name {
				return entry
			}
		}
	case []interface{}:
		for _, item := range v {
			entry, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			if n, _ := entry["name"].(string); n == name {
				return entry
			}
		}
	}
	t.Fatalf("env var %q not found in openwebui.extraEnvVars", name)
	return nil
}

// TestGenerateOverrides_NilConfig ensures that passing a nil config returns an error.
func TestGenerateOverrides_NilConfig(t *testing.T) {
	_, err := GenerateOverrides(nil, DefaultReleaseName, map[string]interface{}{}, nil, nil)
	if err == nil {
		t.Fatal("expected error for nil config, got nil")
	}
}

// TestGenerateOverrides_ProtectedOpenWebUI_OllamaDisabled ensures that
// openwebui.ollama.enabled is always set to false.
func TestGenerateOverrides_ProtectedOpenWebUI_OllamaDisabled(t *testing.T) {
	t.Setenv("LITELLM_MASTER_KEY", "sk-test-key")
	cfg := minimalConfig()
	overrides, err := GenerateOverrides(cfg, DefaultReleaseName, map[string]interface{}{}, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	val := getNestedVal(overrides, "openwebui", "ollama", "enabled")
	if val != false {
		t.Errorf("openwebui.ollama.enabled = %v, want false", val)
	}
}

// TestGenerateOverrides_ProtectedOpenWebUI_WebsocketRedisDisabled ensures that
// openwebui.websocket.redis.enabled is always set to false.
func TestGenerateOverrides_ProtectedOpenWebUI_WebsocketRedisDisabled(t *testing.T) {
	t.Setenv("LITELLM_MASTER_KEY", "sk-test-key")
	cfg := minimalConfig()
	overrides, err := GenerateOverrides(cfg, DefaultReleaseName, map[string]interface{}{}, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	val := getNestedVal(overrides, "openwebui", "websocket", "redis", "enabled")
	if val != false {
		t.Errorf("openwebui.websocket.redis.enabled = %v, want false", val)
	}
}

// TestGenerateOverrides_ProtectedOpenWebUI_DatabaseURLInjected ensures that
// openwebui.extraEnvVars contains DATABASE_URL pointing to the postgresql secret.
func TestGenerateOverrides_ProtectedOpenWebUI_DatabaseURLInjected(t *testing.T) {
	t.Setenv("LITELLM_MASTER_KEY", "sk-test-key")
	cfg := minimalConfig()
	overrides, err := GenerateOverrides(cfg, DefaultReleaseName, map[string]interface{}{}, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	entry := getEnvVar(t, overrides, "DATABASE_URL")
	valueFrom, ok := entry["valueFrom"].(map[string]interface{})
	if !ok {
		t.Fatalf("DATABASE_URL entry missing 'valueFrom' map")
	}
	secretRef, ok := valueFrom["secretKeyRef"].(map[string]interface{})
	if !ok {
		t.Fatalf("DATABASE_URL valueFrom missing 'secretKeyRef' map")
	}
	wantName := "inferencehub-postgresql-secret"
	wantKey := "openwebui-database-url"
	if secretRef["name"] != wantName {
		t.Errorf("DATABASE_URL secretKeyRef.name = %v, want %s", secretRef["name"], wantName)
	}
	if secretRef["key"] != wantKey {
		t.Errorf("DATABASE_URL secretKeyRef.key = %v, want %s", secretRef["key"], wantKey)
	}
}

// TestGenerateOverrides_ProtectedOpenWebUI_OpenAIKeyInjected ensures that
// openwebui.extraEnvVars contains OPENAI_API_KEY pointing to the litellm secret.
func TestGenerateOverrides_ProtectedOpenWebUI_OpenAIKeyInjected(t *testing.T) {
	t.Setenv("LITELLM_MASTER_KEY", "sk-test-key")
	cfg := minimalConfig()
	overrides, err := GenerateOverrides(cfg, DefaultReleaseName, map[string]interface{}{}, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	entry := getEnvVar(t, overrides, "OPENAI_API_KEY")
	valueFrom, ok := entry["valueFrom"].(map[string]interface{})
	if !ok {
		t.Fatalf("OPENAI_API_KEY entry missing 'valueFrom' map")
	}
	secretRef, ok := valueFrom["secretKeyRef"].(map[string]interface{})
	if !ok {
		t.Fatalf("OPENAI_API_KEY valueFrom missing 'secretKeyRef' map")
	}
	wantName := "inferencehub-litellm-secret"
	wantKey := "master-key"
	if secretRef["name"] != wantName {
		t.Errorf("OPENAI_API_KEY secretKeyRef.name = %v, want %s", secretRef["name"], wantName)
	}
	if secretRef["key"] != wantKey {
		t.Errorf("OPENAI_API_KEY secretKeyRef.key = %v, want %s", secretRef["key"], wantKey)
	}
}

// TestGenerateOverrides_ProtectedOpenWebUI_OpenAIBaseURL ensures that
// openwebui.openaiBaseApiUrl points to the LiteLLM service.
func TestGenerateOverrides_ProtectedOpenWebUI_OpenAIBaseURL(t *testing.T) {
	t.Setenv("LITELLM_MASTER_KEY", "sk-test-key")
	cfg := minimalConfig()
	overrides, err := GenerateOverrides(cfg, DefaultReleaseName, map[string]interface{}{}, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	val := getNestedVal(overrides, "openwebui", "openaiBaseApiUrl")
	want := "http://inferencehub-litellm:4000/v1"
	if val != want {
		t.Errorf("openwebui.openaiBaseApiUrl = %v, want %s", val, want)
	}
}

// TestGenerateOverrides_ProtectedLiteLLM_StandaloneDBDisabled ensures that
// litellm.db.deployStandalone is false and litellm.db.useExisting is true.
func TestGenerateOverrides_ProtectedLiteLLM_StandaloneDBDisabled(t *testing.T) {
	t.Setenv("LITELLM_MASTER_KEY", "sk-test-key")
	cfg := minimalConfig()
	overrides, err := GenerateOverrides(cfg, DefaultReleaseName, map[string]interface{}{}, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	deployStandalone := getNestedVal(overrides, "litellm", "db", "deployStandalone")
	if deployStandalone != false {
		t.Errorf("litellm.db.deployStandalone = %v, want false", deployStandalone)
	}
	useExisting := getNestedVal(overrides, "litellm", "db", "useExisting")
	if useExisting != true {
		t.Errorf("litellm.db.useExisting = %v, want true", useExisting)
	}
}

// TestGenerateOverrides_ProtectedLiteLLM_EnvSecretAppended ensures that
// litellm.environmentSecrets contains "inferencehub-litellm-env".
func TestGenerateOverrides_ProtectedLiteLLM_EnvSecretAppended(t *testing.T) {
	t.Setenv("LITELLM_MASTER_KEY", "sk-test-key")
	cfg := minimalConfig()
	overrides, err := GenerateOverrides(cfg, DefaultReleaseName, map[string]interface{}{}, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	litellmMap, ok := overrides["litellm"].(map[string]interface{})
	if !ok {
		t.Fatal("overrides['litellm'] is not a map")
	}
	envSecretsRaw, ok := litellmMap["environmentSecrets"]
	if !ok {
		t.Fatal("litellm missing 'environmentSecrets' key")
	}
	wantSecret := "inferencehub-litellm-env"
	found := false
	switch v := envSecretsRaw.(type) {
	case []string:
		for _, s := range v {
			if s == wantSecret {
				found = true
				break
			}
		}
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok && s == wantSecret {
				found = true
				break
			}
		}
	}
	if !found {
		t.Errorf("litellm.environmentSecrets does not contain %q; got %v", wantSecret, envSecretsRaw)
	}
}

// TestGenerateOverrides_ProtectedLiteLLM_MasterKeySecretRefs ensures that
// litellm.masterkeySecretName and litellm.masterkeySecretKey are set correctly.
func TestGenerateOverrides_ProtectedLiteLLM_MasterKeySecretRefs(t *testing.T) {
	t.Setenv("LITELLM_MASTER_KEY", "sk-test-key")
	cfg := minimalConfig()
	overrides, err := GenerateOverrides(cfg, DefaultReleaseName, map[string]interface{}{}, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	secretName := getNestedVal(overrides, "litellm", "masterkeySecretName")
	if secretName != "inferencehub-litellm-secret" {
		t.Errorf("litellm.masterkeySecretName = %v, want inferencehub-litellm-secret", secretName)
	}
	secretKey := getNestedVal(overrides, "litellm", "masterkeySecretKey")
	if secretKey != "master-key" {
		t.Errorf("litellm.masterkeySecretKey = %v, want master-key", secretKey)
	}
}

// TestGenerateOverrides_ModelList_Bedrock ensures that Bedrock models generate
// the correct proxy_config.model_list entry with bedrock/<model> and aws_region_name.
func TestGenerateOverrides_ModelList_Bedrock(t *testing.T) {
	t.Setenv("LITELLM_MASTER_KEY", "sk-test-key")
	cfg := minimalConfig()
	cfg.Models = config.ModelsConfig{
		Bedrock: []config.BedrockModel{
			{Name: "claude-3", Model: "anthropic.claude-3-sonnet-20240229-v1:0", Region: "us-east-1"},
		},
	}
	overrides, err := GenerateOverrides(cfg, DefaultReleaseName, map[string]interface{}{}, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	litellmMap, ok := overrides["litellm"].(map[string]interface{})
	if !ok {
		t.Fatal("overrides['litellm'] is not a map")
	}
	pc, ok := litellmMap["proxy_config"].(map[string]interface{})
	if !ok {
		t.Fatal("litellm['proxy_config'] is not a map")
	}
	modelList, ok := pc["model_list"].([]map[string]interface{})
	if !ok {
		t.Fatalf("proxy_config['model_list'] has unexpected type %T", pc["model_list"])
	}
	if len(modelList) != 1 {
		t.Fatalf("expected 1 model, got %d", len(modelList))
	}
	entry := modelList[0]
	if entry["model_name"] != "claude-3" {
		t.Errorf("model_name = %v, want claude-3", entry["model_name"])
	}
	params, ok := entry["litellm_params"].(map[string]interface{})
	if !ok {
		t.Fatal("litellm_params is not a map")
	}
	if params["model"] != "bedrock/anthropic.claude-3-sonnet-20240229-v1:0" {
		t.Errorf("model = %v, want bedrock/anthropic.claude-3-sonnet-20240229-v1:0", params["model"])
	}
	if params["aws_region_name"] != "us-east-1" {
		t.Errorf("aws_region_name = %v, want us-east-1", params["aws_region_name"])
	}
}

// TestGenerateOverrides_ModelList_OpenAI ensures that OpenAI models generate the correct entry.
func TestGenerateOverrides_ModelList_OpenAI(t *testing.T) {
	t.Setenv("LITELLM_MASTER_KEY", "sk-test-key")
	cfg := minimalConfig()
	cfg.Models = config.ModelsConfig{
		OpenAI: []config.OpenAIModel{{Name: "gpt-4o", Model: "gpt-4o"}},
	}
	overrides, err := GenerateOverrides(cfg, DefaultReleaseName, map[string]interface{}{}, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	litellmMap := overrides["litellm"].(map[string]interface{})
	pc := litellmMap["proxy_config"].(map[string]interface{})
	modelList := pc["model_list"].([]map[string]interface{})
	if len(modelList) != 1 {
		t.Fatalf("expected 1 model, got %d", len(modelList))
	}
	entry := modelList[0]
	if entry["model_name"] != "gpt-4o" {
		t.Errorf("model_name = %v, want gpt-4o", entry["model_name"])
	}
	params := entry["litellm_params"].(map[string]interface{})
	if params["model"] != "gpt-4o" {
		t.Errorf("model = %v, want gpt-4o", params["model"])
	}
}

// TestGenerateOverrides_ModelList_Ollama ensures that Ollama models generate the correct entry
// with ollama/<model> and api_base.
func TestGenerateOverrides_ModelList_Ollama(t *testing.T) {
	t.Setenv("LITELLM_MASTER_KEY", "sk-test-key")
	cfg := minimalConfig()
	cfg.Models = config.ModelsConfig{
		Ollama: []config.OllamaModel{{Name: "llama3", Model: "llama3", APIBase: "http://host.docker.internal:11434"}},
	}
	overrides, err := GenerateOverrides(cfg, DefaultReleaseName, map[string]interface{}{}, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	litellmMap := overrides["litellm"].(map[string]interface{})
	pc := litellmMap["proxy_config"].(map[string]interface{})
	modelList := pc["model_list"].([]map[string]interface{})
	if len(modelList) != 1 {
		t.Fatalf("expected 1 model, got %d", len(modelList))
	}
	entry := modelList[0]
	if entry["model_name"] != "llama3" {
		t.Errorf("model_name = %v, want llama3", entry["model_name"])
	}
	params := entry["litellm_params"].(map[string]interface{})
	if params["model"] != "ollama/llama3" {
		t.Errorf("model = %v, want ollama/llama3", params["model"])
	}
	if params["api_base"] != "http://host.docker.internal:11434" {
		t.Errorf("api_base = %v, want http://host.docker.internal:11434", params["api_base"])
	}
}

// TestGenerateOverrides_ModelList_MultiProvider ensures that models from different providers
// all appear in the model_list.
func TestGenerateOverrides_ModelList_MultiProvider(t *testing.T) {
	t.Setenv("LITELLM_MASTER_KEY", "sk-test-key")
	cfg := minimalConfig()
	cfg.Models = config.ModelsConfig{
		Bedrock: []config.BedrockModel{
			{Name: "claude-3", Model: "anthropic.claude-3-sonnet-20240229-v1:0", Region: "us-east-1"},
		},
		OpenAI: []config.OpenAIModel{
			{Name: "gpt-4o", Model: "gpt-4o"},
		},
		Ollama: []config.OllamaModel{
			{Name: "llama3", Model: "llama3", APIBase: "http://host.docker.internal:11434"},
		},
	}
	overrides, err := GenerateOverrides(cfg, DefaultReleaseName, map[string]interface{}{}, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	litellmMap := overrides["litellm"].(map[string]interface{})
	pc := litellmMap["proxy_config"].(map[string]interface{})
	modelList := pc["model_list"].([]map[string]interface{})
	if len(modelList) != 3 {
		t.Fatalf("expected 3 models, got %d", len(modelList))
	}
	names := map[string]bool{}
	for _, entry := range modelList {
		if n, ok := entry["model_name"].(string); ok {
			names[n] = true
		}
	}
	for _, want := range []string{"claude-3", "gpt-4o", "llama3"} {
		if !names[want] {
			t.Errorf("model %q not found in model_list", want)
		}
	}
}

// TestGenerateOverrides_PassthroughNotOverriddenByProtected ensures that a non-protected
// user passthrough value (e.g., openwebui.defaultLocale) survives into the output.
func TestGenerateOverrides_PassthroughNotOverriddenByProtected(t *testing.T) {
	t.Setenv("LITELLM_MASTER_KEY", "sk-test-key")
	cfg := minimalConfig()
	cfg.OpenWebUI = map[string]interface{}{
		"defaultLocale": "fr-FR",
	}
	overrides, err := GenerateOverrides(cfg, DefaultReleaseName, map[string]interface{}{}, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	val := getNestedVal(overrides, "openwebui", "defaultLocale")
	if val != "fr-FR" {
		t.Errorf("openwebui.defaultLocale = %v, want fr-FR", val)
	}
}

// TestGenerateOverrides_PassthroughProtectedKeyOverridden ensures that a user setting
// openwebui.ollama.enabled: true in passthrough is overridden to false.
func TestGenerateOverrides_PassthroughProtectedKeyOverridden(t *testing.T) {
	t.Setenv("LITELLM_MASTER_KEY", "sk-test-key")
	cfg := minimalConfig()
	cfg.OpenWebUI = map[string]interface{}{
		"ollama": map[string]interface{}{
			"enabled": true,
		},
	}
	overrides, err := GenerateOverrides(cfg, DefaultReleaseName, map[string]interface{}{}, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	val := getNestedVal(overrides, "openwebui", "ollama", "enabled")
	if val != false {
		t.Errorf("openwebui.ollama.enabled = %v, want false even when passthrough sets true", val)
	}
}

// TestGenerateOverrides_IRSA_AnnotationInjected ensures that when cfg.AWS.LiteLLMRoleARN
// is set, litellm.serviceAccount.annotations has the IRSA annotation.
func TestGenerateOverrides_IRSA_AnnotationInjected(t *testing.T) {
	t.Setenv("LITELLM_MASTER_KEY", "sk-test-key")
	cfg := minimalConfig()
	cfg.AWS = config.AWSConfig{LiteLLMRoleARN: "arn:aws:iam::123456789012:role/LiteLLMRole"}
	overrides, err := GenerateOverrides(cfg, DefaultReleaseName, map[string]interface{}{}, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	annotations := getNestedVal(overrides, "litellm", "serviceAccount", "annotations")
	annotationsMap, ok := annotations.(map[string]interface{})
	if !ok {
		t.Fatalf("litellm.serviceAccount.annotations is not a map, got %T", annotations)
	}
	arn, ok := annotationsMap["eks.amazonaws.com/role-arn"]
	if !ok {
		t.Fatal("IRSA annotation 'eks.amazonaws.com/role-arn' not found")
	}
	if arn != "arn:aws:iam::123456789012:role/LiteLLMRole" {
		t.Errorf("IRSA annotation value = %v, want arn:aws:iam::123456789012:role/LiteLLMRole", arn)
	}
}

// TestGenerateOverrides_IRSAAnnotation_NotInjectedWhenEmpty ensures that when
// cfg.AWS.LiteLLMRoleARN is empty, litellm.serviceAccount.annotations is not set.
func TestGenerateOverrides_IRSAAnnotation_NotInjectedWhenEmpty(t *testing.T) {
	t.Setenv("LITELLM_MASTER_KEY", "sk-test-key")
	cfg := minimalConfig()
	cfg.AWS = config.AWSConfig{LiteLLMRoleARN: ""}
	overrides, err := GenerateOverrides(cfg, DefaultReleaseName, map[string]interface{}{}, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	annotations := getNestedVal(overrides, "litellm", "serviceAccount", "annotations")
	if annotations != nil {
		t.Errorf("litellm.serviceAccount.annotations should be nil when IRSA ARN is empty, got %v", annotations)
	}
}

// TestGenerateOverrides_VersionOverride_OpenWebUI ensures that setting
// cfg.Versions.OpenWebUI overrides the image tag.
func TestGenerateOverrides_VersionOverride_OpenWebUI(t *testing.T) {
	t.Setenv("LITELLM_MASTER_KEY", "sk-test-key")
	cfg := minimalConfig()
	cfg.Versions.OpenWebUI = "0.9.0"
	overrides, err := GenerateOverrides(cfg, DefaultReleaseName, map[string]interface{}{}, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tag := getNestedVal(overrides, "openwebui", "image", "tag")
	if tag != "0.9.0" {
		t.Errorf("openwebui.image.tag = %v, want 0.9.0", tag)
	}
}

// TestGenerateOverrides_EmptyModelList ensures that a config with no models
// does not panic and returns an empty model_list.
func TestGenerateOverrides_EmptyModelList(t *testing.T) {
	t.Setenv("LITELLM_MASTER_KEY", "sk-test-key")
	cfg := minimalConfig()
	cfg.Models = config.ModelsConfig{}
	overrides, err := GenerateOverrides(cfg, DefaultReleaseName, map[string]interface{}{}, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	litellmMap, ok := overrides["litellm"].(map[string]interface{})
	if !ok {
		t.Fatal("overrides['litellm'] is not a map")
	}
	pc, ok := litellmMap["proxy_config"].(map[string]interface{})
	if !ok {
		t.Fatal("litellm['proxy_config'] is not a map")
	}
	modelList := pc["model_list"].([]map[string]interface{})
	if len(modelList) != 0 {
		t.Errorf("expected empty model_list, got %d entries", len(modelList))
	}
}

// TestGenerateOverrides_FullnameOverride ensures that the release name is correctly
// used in the LiteLLM service URL.
func TestGenerateOverrides_FullnameOverride(t *testing.T) {
	t.Setenv("LITELLM_MASTER_KEY", "sk-test-key")
	cfg := minimalConfig()
	overrides, err := GenerateOverrides(cfg, "inferencehub", map[string]interface{}{}, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	val := getNestedVal(overrides, "openwebui", "openaiBaseApiUrl")
	want := "http://inferencehub-litellm:4000/v1"
	if val != want {
		t.Errorf("openaiBaseApiUrl = %v, want %s", val, want)
	}
}

// TestGenerateOverrides_WebSearch_InCluster_SearXNGEnabled ensures that when
// webSearch.enabled=true and external is disabled, searxng.enabled=true and
// ENABLE_RAG_WEB_SEARCH env var is set.
func TestGenerateOverrides_WebSearch_InCluster_SearXNGEnabled(t *testing.T) {
	t.Setenv("LITELLM_MASTER_KEY", "sk-test-key")
	t.Setenv("SEARXNG_SECRET_KEY", "test-searxng-secret")
	cfg := minimalConfig()
	cfg.WebSearch = config.WebSearchConfig{
		Enabled: true,
		// External.Enabled is false by default — uses in-cluster SearXNG
	}
	overrides, err := GenerateOverrides(cfg, DefaultReleaseName, map[string]interface{}{}, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	searxngEnabled := getNestedVal(overrides, "searxng", "enabled")
	if searxngEnabled != true {
		t.Errorf("searxng.enabled = %v, want true", searxngEnabled)
	}
	entry := getEnvVar(t, overrides, "ENABLE_RAG_WEB_SEARCH")
	if entry["value"] != "true" {
		t.Errorf("ENABLE_RAG_WEB_SEARCH value = %v, want true", entry["value"])
	}
}

// TestGenerateOverrides_RedisURL_InCluster ensures that the websocket URL uses
// the in-cluster Redis hostname with no password.
func TestGenerateOverrides_RedisURL_InCluster(t *testing.T) {
	t.Setenv("LITELLM_MASTER_KEY", "sk-test-key")
	cfg := minimalConfig()
	// No password set — plain URL expected
	overrides, err := GenerateOverrides(cfg, DefaultReleaseName, map[string]interface{}{}, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wsURL := getNestedVal(overrides, "openwebui", "websocket", "url")
	want := "redis://inferencehub-redis-openwebui:6379/0"
	if wsURL != want {
		t.Errorf("openwebui.websocket.url = %v, want %s", wsURL, want)
	}
}
