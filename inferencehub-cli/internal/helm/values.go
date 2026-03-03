package helm

import (
	"context"
	"fmt"
	"os"

	"github.com/Vinay-Venkatesh/inferencehub-cli/internal/config"
	"github.com/Vinay-Venkatesh/inferencehub-cli/internal/k8s"
	"sigs.k8s.io/yaml"
)

// DefaultReleaseName is the Helm release name used by InferenceHub.
// It determines internal service URLs and secret names.
const DefaultReleaseName = "inferencehub"

// LoadValuesFile reads a YAML Helm values file and returns it as a map.
// Returns an empty map if path is empty (no-op).
func LoadValuesFile(path string) (map[string]interface{}, error) {
	if path == "" {
		return map[string]interface{}{}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read values file %s: %w", path, err)
	}
	values := map[string]interface{}{}
	if err := yaml.Unmarshal(data, &values); err != nil {
		return nil, fmt.Errorf("failed to parse values file %s: %w", path, err)
	}
	return values, nil
}

// MergeValues deep-merges src on top of dst. Values in src take precedence.
// When both sides have a map at the same key, they are merged recursively.
// Returns a new map without modifying dst or src.
func MergeValues(dst, src map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(dst))
	for k, v := range dst {
		result[k] = v
	}
	for k, sv := range src {
		if dv, ok := result[k]; ok {
			if dm, ok := dv.(map[string]interface{}); ok {
				if sm, ok := sv.(map[string]interface{}); ok {
					result[k] = MergeValues(dm, sm)
					continue
				}
			}
		}
		result[k] = sv
	}
	return result
}

// GenerateOverrides converts a Config into a Helm values override map.
// releaseName is the Helm release name used to compute internal service URLs.
// The overrides are merged on top of the chart's base values.yaml (and any -f values files).
//
// IMPORTANT: the Helm Go SDK treats map keys as literal strings — it does NOT expand
// dot-notation into nested paths. All values must be expressed as proper nested maps.
func GenerateOverrides(cfg *config.Config, releaseName string, ctx context.Context, k8sClient *k8s.Client) (map[string]interface{}, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	overrides := map[string]interface{}{}
	fullname := fmt.Sprintf("%s-inferencehub", releaseName)

	// Global namespace
	setNested(overrides, cfg.EffectiveNamespace(), "global", "namespace")

	// Component image versions for non-subchart components
	if cfg.Versions.PostgreSQL != "" {
		setNested(overrides, cfg.Versions.PostgreSQL, "postgresql", "image", "tag")
	}
	if cfg.Versions.Redis != "" {
		setNested(overrides, cfg.Versions.Redis, "redis", "openwebui", "image", "tag")
		setNested(overrides, cfg.Versions.Redis, "redis", "litellm", "image", "tag")
	}

	// ── OpenWebUI subchart values ──────────────────────────────────────────────
	// Start with user's passthrough values, then apply protected injections on top.
	owValues := map[string]interface{}{}
	if cfg.OpenWebUI != nil {
		owValues = deepCopyMap(cfg.OpenWebUI)
	}

	// Protected: disable subcharts that InferenceHub replaces or doesn't provision.
	// ollama is always disabled — users who want Ollama use models.ollama with an external URL.
	// pipelines is NOT forced here — users can enable it via the openwebui: passthrough.
	setNested(owValues, false, "ollama", "enabled")
	setNested(owValues, false, "websocket", "redis", "enabled")

	// Protected: point websocket to the OpenWebUI Redis instance
	owRedisHost := inferredOpenWebUIRedisHost(cfg, releaseName)
	owRedisPort := inferredRedisPort(cfg.Redis.OpenWebUI)
	owRedisPassword := cfg.Redis.OpenWebUI.Password
	if owRedisPassword == "" {
		owRedisPassword = os.Getenv("OPENWEBUI_REDIS_PASSWORD")
	}
	var redisURL string
	if owRedisPassword != "" {
		redisURL = fmt.Sprintf("redis://:%s@%s:%s/0", owRedisPassword, owRedisHost, owRedisPort)
	} else {
		redisURL = fmt.Sprintf("redis://%s:%s/0", owRedisHost, owRedisPort)
	}
	setNested(owValues, redisURL, "websocket", "url")

	// Protected: point OpenWebUI at LiteLLM
	litellmPort := 4000
	setNested(owValues, fmt.Sprintf("http://%s-litellm:%d/v1", releaseName, litellmPort), "openaiBaseApiUrl")

	// Protected: inject DATABASE_URL and OPENAI_API_KEY via extraEnvVars
	inferencehubEnvVars := []map[string]interface{}{
		{
			"name": "DATABASE_URL",
			"valueFrom": map[string]interface{}{
				"secretKeyRef": map[string]interface{}{
					"name": fullname + "-postgresql-secret",
					"key":  "openwebui-database-url",
				},
			},
		},
		{
			"name": "OPENAI_API_KEY",
			"valueFrom": map[string]interface{}{
				"secretKeyRef": map[string]interface{}{
					"name": fullname + "-litellm-secret",
					"key":  "master-key",
				},
			},
		},
	}
	owValues["extraEnvVars"] = mergeEnvVarLists(
		extractEnvVarList(owValues),
		inferencehubEnvVars,
	)

	// Protected: image tag (from versions.openwebui)
	if cfg.Versions.OpenWebUI != "" {
		setNested(owValues, cfg.Versions.OpenWebUI, "image", "tag")
	}

	setNested(overrides, owValues, "openwebui")

	// ── LiteLLM subchart values ────────────────────────────────────────────────
	// Start with user's passthrough values, then apply protected injections on top.
	litellmValues := map[string]interface{}{}
	if cfg.LiteLLM != nil {
		litellmValues = deepCopyMap(cfg.LiteLLM)
	}

	// Protected: disable bundled database and Redis
	setNested(litellmValues, false, "db", "deployStandalone")
	setNested(litellmValues, true, "db", "useExisting")
	setNested(litellmValues, false, "redis", "enabled")

	// Protected: master key value (used by parent chart's templates/litellm/secret.yaml)
	setNested(litellmValues, os.Getenv("LITELLM_MASTER_KEY"), "masterKey")

	// Protected: master key secret references (used by litellm-helm subchart)
	setNested(litellmValues, fullname+"-litellm-secret", "masterkeySecretName")
	setNested(litellmValues, "master-key", "masterkeySecretKey")

	// Protected: inject InferenceHub's wiring secret (DATABASE_URL, REDIS_*)
	litellmValues["environmentSecrets"] = mergeStringLists(
		extractStringList(litellmValues, "environmentSecrets"),
		[]string{fullname + "-litellm-env"},
	)

	// Protected: model list from inferencehub.yaml models: section
	existingPC, _ := litellmValues["proxy_config"].(map[string]interface{})
	if existingPC == nil {
		existingPC = map[string]interface{}{}
	}
	existingPC["model_list"] = buildModelList(cfg.Models)
	// Protected: master key reference in general_settings
	gs, _ := existingPC["general_settings"].(map[string]interface{})
	if gs == nil {
		gs = map[string]interface{}{}
	}
	gs["master_key"] = "os.environ/LITELLM_MASTER_KEY"
	existingPC["general_settings"] = gs
	litellmValues["proxy_config"] = existingPC

	// Protected: IRSA annotation
	if cfg.AWS.LiteLLMRoleARN != "" {
		setNested(litellmValues, map[string]interface{}{
			"eks.amazonaws.com/role-arn": cfg.AWS.LiteLLMRoleARN,
		}, "serviceAccount", "annotations")
	}

	// Protected: image tag (from versions.litellm)
	if cfg.Versions.LiteLLM != "" {
		setNested(litellmValues, cfg.Versions.LiteLLM, "image", "tag")
	}

	// Protected: Langfuse observability
	if cfg.Observability.Enabled {
		setNested(litellmValues, cfg.Observability.Langfuse.Host, "envVars", "LANGFUSE_HOST")
		setNested(litellmValues, cfg.Observability.Langfuse.PublicKey, "envVars", "LANGFUSE_PUBLIC_KEY")
		setNested(litellmValues, cfg.Observability.Langfuse.SecretKey, "envVars", "LANGFUSE_SECRET_KEY")
	}

	setNested(overrides, litellmValues, "litellm")

	// ── PostgreSQL ─────────────────────────────────────────────────────────────
	if cfg.PostgreSQL.IsExternal() {
		setNested(overrides, false, "postgresql", "enabled")
		setNested(overrides, true, "postgresql", "external", "enabled")
		if cfg.PostgreSQL.OpenWebUIConnectionString != "" {
			setNested(overrides, cfg.PostgreSQL.OpenWebUIConnectionString, "postgresql", "external", "openwebuiConnectionString")
		}
		if cfg.PostgreSQL.LiteLLMConnectionString != "" {
			setNested(overrides, cfg.PostgreSQL.LiteLLMConnectionString, "postgresql", "external", "litellmConnectionString")
		}
	} else {
		pgPassword := cfg.PostgreSQL.Password
		if pgPassword == "" {
			pgPassword = os.Getenv("POSTGRES_PASSWORD")
		}
		if pgPassword != "" {
			setNested(overrides, pgPassword, "postgresql", "auth", "password")
		}
		if sc, err := detectStorageClass(ctx, k8sClient); err == nil && sc != "" {
			setNested(overrides, sc, "postgresql", "persistence", "storageClass")
		}
	}

	// ── Redis (per-app) ────────────────────────────────────────────────────────
	sc, _ := detectStorageClass(ctx, k8sClient)

	// OpenWebUI Redis
	if cfg.Redis.OpenWebUI.IsExternal() {
		setNested(overrides, false, "redis", "openwebui", "enabled")
		setNested(overrides, true, "redis", "openwebui", "external", "enabled")
		setNested(overrides, extractHost(cfg.Redis.OpenWebUI.URL), "redis", "openwebui", "external", "host")
		if cfg.Redis.OpenWebUI.Password != "" {
			setNested(overrides, cfg.Redis.OpenWebUI.Password, "redis", "openwebui", "external", "password")
		}
	} else {
		owRedisPass := cfg.Redis.OpenWebUI.Password
		if owRedisPass == "" {
			owRedisPass = os.Getenv("OPENWEBUI_REDIS_PASSWORD")
		}
		if owRedisPass != "" {
			setNested(overrides, owRedisPass, "redis", "openwebui", "auth", "password")
		}
		if sc != "" {
			setNested(overrides, sc, "redis", "openwebui", "persistence", "storageClass")
		}
	}

	// LiteLLM Redis
	if cfg.Redis.LiteLLM.IsExternal() {
		setNested(overrides, false, "redis", "litellm", "enabled")
		setNested(overrides, true, "redis", "litellm", "external", "enabled")
		setNested(overrides, extractHost(cfg.Redis.LiteLLM.URL), "redis", "litellm", "external", "host")
		if cfg.Redis.LiteLLM.Password != "" {
			setNested(overrides, cfg.Redis.LiteLLM.Password, "redis", "litellm", "external", "password")
		}
	} else {
		litellmRedisPass := cfg.Redis.LiteLLM.Password
		if litellmRedisPass == "" {
			litellmRedisPass = os.Getenv("LITELLM_REDIS_PASSWORD")
		}
		if litellmRedisPass != "" {
			setNested(overrides, litellmRedisPass, "redis", "litellm", "auth", "password")
		}
		if sc != "" {
			setNested(overrides, sc, "redis", "litellm", "persistence", "storageClass")
		}
	}

	// ── Networking ─────────────────────────────────────────────────────────────
	setNested(overrides, cfg.Domain, "networking", "gatewayAPI", "hostname")
	setNested(overrides, cfg.Gateway.Name, "networking", "gatewayAPI", "gatewayRef", "name")
	setNested(overrides, cfg.Gateway.Namespace, "networking", "gatewayAPI", "gatewayRef", "namespace")
	setNested(overrides, cfg.IssuerType(), "networking", "gatewayAPI", "tls", "issuerRef")

	// ── Observability ──────────────────────────────────────────────────────────
	if cfg.Observability.Enabled {
		setNested(overrides, true, "observability", "langfuse", "enabled")
		setNested(overrides, cfg.Observability.Langfuse.Host, "observability", "langfuse", "host")
		setNested(overrides, cfg.Observability.Langfuse.PublicKey, "observability", "langfuse", "publicKey")
		setNested(overrides, cfg.Observability.Langfuse.SecretKey, "observability", "langfuse", "secretKey")
	}

	return overrides, nil
}

// setNested sets value at the given path inside m, creating intermediate maps as needed.
func setNested(m map[string]interface{}, value interface{}, path ...string) {
	for _, key := range path[:len(path)-1] {
		next, ok := m[key]
		if !ok {
			next = map[string]interface{}{}
			m[key] = next
		}
		nextMap, ok := next.(map[string]interface{})
		if !ok {
			nextMap = map[string]interface{}{}
			m[key] = nextMap
		}
		m = nextMap
	}
	m[path[len(path)-1]] = value
}

// buildModelList converts the grouped ModelsConfig into LiteLLM model_list format.
func buildModelList(models config.ModelsConfig) []map[string]interface{} {
	var list []map[string]interface{}

	for _, m := range models.Bedrock {
		list = append(list, map[string]interface{}{
			"model_name": m.Name,
			"litellm_params": map[string]interface{}{
				"model":           fmt.Sprintf("bedrock/%s", m.Model),
				"aws_region_name": m.Region,
			},
		})
	}

	for _, m := range models.OpenAI {
		list = append(list, map[string]interface{}{
			"model_name": m.Name,
			"litellm_params": map[string]interface{}{
				"model": m.Model,
			},
		})
	}

	for _, m := range models.Ollama {
		list = append(list, map[string]interface{}{
			"model_name": m.Name,
			"litellm_params": map[string]interface{}{
				"model":    fmt.Sprintf("ollama/%s", m.Model),
				"api_base": m.APIBase,
			},
		})
	}

	for _, m := range models.Azure {
		params := map[string]interface{}{
			"model":    fmt.Sprintf("azure/%s", m.Model),
			"api_base": m.APIBase,
		}
		if m.APIVersion != "" {
			params["api_version"] = m.APIVersion
		}
		list = append(list, map[string]interface{}{
			"model_name":     m.Name,
			"litellm_params": params,
		})
	}

	return list
}

// detectStorageClass finds the best available storage class in the cluster.
func detectStorageClass(ctx context.Context, k8sClient *k8s.Client) (string, error) {
	if ctx == nil || k8sClient == nil {
		return "", nil
	}
	sc, err := k8sClient.GetDefaultStorageClass(ctx)
	if err != nil {
		return "", err
	}
	return sc, nil
}

// extractHost parses just the host from a URL-ish string.
func extractHost(url string) string {
	for _, prefix := range []string{"redis://", "rediss://", "http://", "https://"} {
		if len(url) > len(prefix) && url[:len(prefix)] == prefix {
			url = url[len(prefix):]
		}
	}
	for i, c := range url {
		if c == '/' {
			return url[:i]
		}
	}
	return url
}

// inferredOpenWebUIRedisHost returns the Redis hostname for the OpenWebUI Redis instance.
func inferredOpenWebUIRedisHost(cfg *config.Config, releaseName string) string {
	if cfg.Redis.OpenWebUI.IsExternal() {
		return extractHost(cfg.Redis.OpenWebUI.URL)
	}
	return fmt.Sprintf("%s-inferencehub-redis-openwebui", releaseName)
}

// inferredRedisPort returns the Redis port string from a RedisAppConfig URL.
func inferredRedisPort(r config.RedisAppConfig) string {
	if r.IsExternal() && r.URL != "" {
		host := extractHost(r.URL)
		for i := len(host) - 1; i >= 0; i-- {
			if host[i] == ':' {
				return host[i+1:]
			}
		}
	}
	return "6379"
}

// mergeEnvVarLists merges two env var lists. Entries in priority override entries in base
// with the same name. Entries unique to priority are appended.
func mergeEnvVarLists(base, priority []map[string]interface{}) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(base)+len(priority))
	priorityNames := map[string]bool{}
	for _, e := range priority {
		if name, ok := e["name"].(string); ok {
			priorityNames[name] = true
		}
	}
	for _, e := range base {
		name, _ := e["name"].(string)
		if !priorityNames[name] {
			result = append(result, e)
		}
	}
	result = append(result, priority...)
	return result
}

// mergeStringLists returns a deduplicated concatenation of two string slices.
func mergeStringLists(base, additions []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(base)+len(additions))
	for _, s := range base {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	for _, s := range additions {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

// extractEnvVarList safely extracts []map[string]interface{} from a values map at "extraEnvVars".
func extractEnvVarList(m map[string]interface{}) []map[string]interface{} {
	raw, ok := m["extraEnvVars"]
	if !ok {
		return nil
	}
	switch v := raw.(type) {
	case []map[string]interface{}:
		return v
	case []interface{}:
		result := make([]map[string]interface{}, 0, len(v))
		for _, item := range v {
			if entry, ok := item.(map[string]interface{}); ok {
				result = append(result, entry)
			}
		}
		return result
	}
	return nil
}

// extractStringList safely extracts []string from a values map at the given key.
func extractStringList(m map[string]interface{}, key string) []string {
	raw, ok := m[key]
	if !ok {
		return nil
	}
	switch v := raw.(type) {
	case []string:
		return v
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
}

// deepCopyMap returns a deep copy of a map[string]interface{} to avoid mutating user config.
func deepCopyMap(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return nil
	}
	result := make(map[string]interface{}, len(m))
	for k, v := range m {
		switch val := v.(type) {
		case map[string]interface{}:
			result[k] = deepCopyMap(val)
		case []interface{}:
			cp := make([]interface{}, len(val))
			copy(cp, val)
			result[k] = cp
		default:
			result[k] = v
		}
	}
	return result
}
