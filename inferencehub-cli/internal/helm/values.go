package helm

import (
	"context"
	"fmt"
	"os"

	"github.com/Vinay-Venkatesh/inferencehub-cli/internal/config"
	"github.com/Vinay-Venkatesh/inferencehub-cli/internal/k8s"
	"sigs.k8s.io/yaml"
)

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
// The overrides are merged on top of the chart's base values.yaml (and any -f values files).
//
// IMPORTANT: the Helm Go SDK treats map keys as literal strings — it does NOT expand
// dot-notation into nested paths. All values must be expressed as proper nested maps.
func GenerateOverrides(cfg *config.Config, ctx context.Context, k8sClient *k8s.Client) (map[string]interface{}, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	overrides := map[string]interface{}{}

	// Global namespace
	setNested(overrides, cfg.EffectiveNamespace(), "global", "namespace")

	// Component image versions (only when explicitly provided in config)
	if cfg.Versions.OpenWebUI != "" {
		setNested(overrides, cfg.Versions.OpenWebUI, "openwebui", "image", "tag")
	}
	if cfg.Versions.LiteLLM != "" {
		setNested(overrides, cfg.Versions.LiteLLM, "litellm", "image", "tag")
	}
	if cfg.Versions.PostgreSQL != "" {
		setNested(overrides, cfg.Versions.PostgreSQL, "postgresql", "image", "tag")
	}
	if cfg.Versions.Redis != "" {
		setNested(overrides, cfg.Versions.Redis, "redis", "image", "tag")
	}

	// LiteLLM master key + model list
	setNested(overrides, os.Getenv("LITELLM_MASTER_KEY"), "litellm", "masterKey")
	setNested(overrides, buildModelList(cfg.Models), "litellm", "models")

	// PostgreSQL
	if cfg.PostgreSQL.IsExternal() {
		setNested(overrides, false, "postgresql", "enabled")
		setNested(overrides, true, "postgresql", "external", "enabled")
		setNested(overrides, buildPostgresConnStr(cfg.PostgreSQL), "postgresql", "external", "connectionString")
	} else {
		// In-cluster PostgreSQL: pass password so the container and Secret are initialised.
		// Source: cfg.PostgreSQL.Password (set via ${POSTGRES_PASSWORD} in config) or the env var directly.
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

	// Redis
	if cfg.Redis.IsExternal() {
		setNested(overrides, false, "redis", "enabled")
		setNested(overrides, true, "redis", "external", "enabled")
		setNested(overrides, extractHost(cfg.Redis.URL), "redis", "external", "host")
		setNested(overrides, cfg.Redis.Password, "redis", "external", "password")
	} else {
		// In-cluster Redis: pass password if provided (Redis runs without auth when empty).
		redisPassword := cfg.Redis.Password
		if redisPassword == "" {
			redisPassword = os.Getenv("REDIS_PASSWORD")
		}
		if redisPassword != "" {
			setNested(overrides, redisPassword, "redis", "auth", "password")
		}
		if sc, err := detectStorageClass(ctx, k8sClient); err == nil && sc != "" {
			setNested(overrides, sc, "redis", "persistence", "storageClass")
		}
	}

	// Networking
	setNested(overrides, cfg.Domain, "networking", "gatewayAPI", "hostname")
	setNested(overrides, cfg.Gateway.Name, "networking", "gatewayAPI", "gatewayRef", "name")
	setNested(overrides, cfg.Gateway.Namespace, "networking", "gatewayAPI", "gatewayRef", "namespace")
	setNested(overrides, cfg.IssuerType(), "networking", "gatewayAPI", "tls", "issuerRef")

	// AWS IRSA annotation — annotates the LiteLLM service account so pods can call Bedrock
	if cfg.AWS.LiteLLMRoleARN != "" {
		setNested(overrides, map[string]interface{}{
			"eks.amazonaws.com/role-arn": cfg.AWS.LiteLLMRoleARN,
		}, "litellm", "serviceAccount", "annotations")
	}

	// Observability
	if cfg.Observability.Enabled {
		setNested(overrides, true, "observability", "langfuse", "enabled")
		setNested(overrides, cfg.Observability.Langfuse.Host, "observability", "langfuse", "host")
		setNested(overrides, cfg.Observability.Langfuse.PublicKey, "observability", "langfuse", "publicKey")
		setNested(overrides, cfg.Observability.Langfuse.SecretKey, "observability", "langfuse", "secretKey")
	}

	return overrides, nil
}

// setNested sets value at the given path inside m, creating intermediate maps as needed.
// Example: setNested(m, "v1.2.3", "openwebui", "image", "tag")
// produces: m["openwebui"]["image"]["tag"] = "v1.2.3"
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
// Returns empty string if detection fails (chart defaults will apply).
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

// buildPostgresConnStr constructs a PostgreSQL connection string.
func buildPostgresConnStr(pg config.DatastoreConfig) string {
	if pg.URL != "" {
		// If URL already looks like a full connection string, use it directly
		if len(pg.URL) > 13 && pg.URL[:13] == "postgresql://" {
			return pg.URL
		}
		// Otherwise treat as host and build the string
		return fmt.Sprintf("postgresql://%s:%s@%s/inferencehub", pg.Username, pg.Password, pg.URL)
	}
	return ""
}

// extractHost parses just the host from a URL-ish string.
func extractHost(url string) string {
	// Strip common schemes
	for _, prefix := range []string{"redis://", "rediss://", "http://", "https://"} {
		if len(url) > len(prefix) && url[:len(prefix)] == prefix {
			url = url[len(prefix):]
		}
	}
	// Remove path
	for i, c := range url {
		if c == '/' {
			return url[:i]
		}
	}
	return url
}
