package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Validate checks all required fields and formats in the Config.
// Returns an error describing the first validation failure found.
func Validate(cfg *Config) error {
	if cfg.ClusterName == "" {
		return fmt.Errorf("clusterName is required")
	}

	if cfg.Domain == "" {
		return fmt.Errorf("domain is required")
	}
	if !isValidDomain(cfg.Domain) {
		return fmt.Errorf("domain %q is not a valid FQDN", cfg.Domain)
	}

	if cfg.Environment == "" {
		return fmt.Errorf("environment is required (e.g., staging, production)")
	}

	if !isValidK8sName(cfg.EffectiveNamespace()) {
		return fmt.Errorf("namespace %q must be a valid Kubernetes name (lowercase alphanumeric and hyphens)", cfg.EffectiveNamespace())
	}

	if cfg.Gateway.Name == "" {
		return fmt.Errorf("gateway.name is required")
	}
	if cfg.Gateway.Namespace == "" {
		return fmt.Errorf("gateway.namespace is required")
	}

	// LITELLM_MASTER_KEY must be set as an env var
	masterKey := os.Getenv("LITELLM_MASTER_KEY")
	if masterKey == "" {
		return fmt.Errorf("LITELLM_MASTER_KEY environment variable is required")
	}
	if !strings.HasPrefix(masterKey, "sk-") {
		return fmt.Errorf("LITELLM_MASTER_KEY must start with 'sk-'")
	}

	// In-cluster PostgreSQL requires POSTGRES_PASSWORD to initialise the database
	if !cfg.PostgreSQL.IsExternal() {
		pgPassword := cfg.PostgreSQL.Password
		if pgPassword == "" {
			pgPassword = os.Getenv("POSTGRES_PASSWORD")
		}
		if pgPassword == "" {
			return fmt.Errorf("POSTGRES_PASSWORD is required for in-cluster PostgreSQL — set it in your .env file or as an environment variable")
		}
	}

	// At least one model must be configured
	if !cfg.Models.HasModels() {
		return fmt.Errorf("at least one model must be configured under models.bedrock, models.openai, models.ollama, or models.azure")
	}

	// Validate each model group
	for i, m := range cfg.Models.Bedrock {
		if m.Name == "" {
			return fmt.Errorf("models.bedrock[%d].name is required", i)
		}
		if m.Model == "" {
			return fmt.Errorf("models.bedrock[%d].model is required", i)
		}
		if m.Region == "" {
			return fmt.Errorf("models.bedrock[%d].region is required for Bedrock models", i)
		}
	}

	for i, m := range cfg.Models.OpenAI {
		if m.Name == "" {
			return fmt.Errorf("models.openai[%d].name is required", i)
		}
		if m.Model == "" {
			return fmt.Errorf("models.openai[%d].model is required", i)
		}
	}

	for i, m := range cfg.Models.Ollama {
		if m.Name == "" {
			return fmt.Errorf("models.ollama[%d].name is required", i)
		}
		if m.Model == "" {
			return fmt.Errorf("models.ollama[%d].model is required", i)
		}
		if m.APIBase == "" {
			return fmt.Errorf("models.ollama[%d].apiBase is required (e.g., http://host.docker.internal:11434)", i)
		}
	}

	for i, m := range cfg.Models.Azure {
		if m.Name == "" {
			return fmt.Errorf("models.azure[%d].name is required", i)
		}
		if m.Model == "" {
			return fmt.Errorf("models.azure[%d].model is required", i)
		}
		if m.APIBase == "" {
			return fmt.Errorf("models.azure[%d].apiBase is required for Azure models", i)
		}
	}

	// Validate external PostgreSQL config
	if cfg.PostgreSQL.IsExternal() {
		if cfg.PostgreSQL.Username == "" {
			return fmt.Errorf("postgresql.username is required when postgresql.url is set")
		}
		if cfg.PostgreSQL.Password == "" {
			return fmt.Errorf("postgresql.password is required when postgresql.url is set")
		}
	}

	// Validate external Redis config (per-app)
	if cfg.Redis.OpenWebUI.IsExternal() {
		if cfg.Redis.OpenWebUI.Password == "" {
			return fmt.Errorf("redis.openwebui.password is required when redis.openwebui.url is set")
		}
	}
	if cfg.Redis.LiteLLM.IsExternal() {
		if cfg.Redis.LiteLLM.Password == "" {
			return fmt.Errorf("redis.litellm.password is required when redis.litellm.url is set")
		}
	}

	// Validate observability config
	if cfg.Observability.Enabled {
		if cfg.Observability.Langfuse.PublicKey == "" {
			return fmt.Errorf("observability.langfuse.publicKey is required when observability is enabled (set LANGFUSE_PUBLIC_KEY)")
		}
		if cfg.Observability.Langfuse.SecretKey == "" {
			return fmt.Errorf("observability.langfuse.secretKey is required when observability is enabled (set LANGFUSE_SECRET_KEY)")
		}
	}

	return nil
}

// ValidateAndWarn validates the config and returns non-fatal warnings alongside errors.
func ValidateAndWarn(cfg *Config) (error, []string) {
	var warnings []string

	if err := Validate(cfg); err != nil {
		return err, warnings
	}

	if cfg.IssuerType() == "letsencrypt-staging" {
		warnings = append(warnings, fmt.Sprintf("environment=%q uses Let's Encrypt staging - TLS certificates will not be trusted by browsers", cfg.Environment))
	}

	if len(cfg.Models.Bedrock) > 0 && cfg.AWS.LiteLLMRoleARN == "" {
		warnings = append(warnings, "Bedrock models configured but aws.litellmRoleArn is not set — LiteLLM will fall back to the node instance role, which may not have Bedrock permissions")
	}

	if !cfg.PostgreSQL.IsExternal() {
		warnings = append(warnings, "Using in-cluster PostgreSQL. For production, consider an external managed database (e.g., RDS)")
	}

	if !cfg.Redis.OpenWebUI.IsExternal() {
		warnings = append(warnings, "Using in-cluster Redis for OpenWebUI. For production, consider an external managed cache (e.g., ElastiCache)")
	}
	if !cfg.Redis.LiteLLM.IsExternal() {
		warnings = append(warnings, "Using in-cluster Redis for LiteLLM. For production, consider an external managed cache (e.g., ElastiCache)")
	}

	// Warn if user is setting protected OpenWebUI keys
	if cfg.OpenWebUI != nil {
		owProtected := []string{"openaiBaseApiUrl", "openaiApiKey"}
		for _, k := range owProtected {
			if _, set := cfg.OpenWebUI[k]; set {
				warnings = append(warnings, fmt.Sprintf(
					"openwebui.%s is managed by InferenceHub and will be overridden — remove it from inferencehub.yaml", k))
			}
		}

		// Warn if user enables the Ollama subchart — it will be forced off
		if v, set := cfg.OpenWebUI["ollama"]; set {
			if ollamaMap, ok := v.(map[string]interface{}); ok {
				if enabled, ok := ollamaMap["enabled"].(bool); ok && enabled {
					warnings = append(warnings, "openwebui.ollama.enabled: true will be overridden to false — InferenceHub disables the Ollama subchart. To use Ollama, add it under models.ollama with an external apiBase URL instead")
				}
			}
		}

		// Warn about protected websocket fields
		if v, set := cfg.OpenWebUI["websocket"]; set {
			if wsMap, ok := v.(map[string]interface{}); ok {
				// websocket.redis.enabled: true is forced off
				if redisMap, ok := wsMap["redis"].(map[string]interface{}); ok {
					if enabled, ok := redisMap["enabled"].(bool); ok && enabled {
						warnings = append(warnings, "openwebui.websocket.redis.enabled: true will be overridden to false — InferenceHub provides a dedicated Redis for OpenWebUI websockets (redis.openwebui). The entire websocket.redis: block (image, resources, labels) is ignored")
					}
				}
				// websocket.url is always computed from InferenceHub's Redis config
				if url, ok := wsMap["url"].(string); ok && url != "" {
					warnings = append(warnings, fmt.Sprintf(
						"openwebui.websocket.url %q will be overridden — InferenceHub computes this from your redis: config. To use a different Redis, set redis.external in inferencehub.yaml", url))
				}
			}
		}
	}

	// Warn if user is setting protected LiteLLM keys
	if cfg.LiteLLM != nil {
		litellmProtected := []string{"masterkeySecretName", "masterkeySecretKey"}
		for _, k := range litellmProtected {
			if _, set := cfg.LiteLLM[k]; set {
				warnings = append(warnings, fmt.Sprintf(
					"litellm.%s is managed by InferenceHub and will be overridden — remove it from inferencehub.yaml", k))
			}
		}

		// Warn if user enables the bundled Redis subchart — it will be forced off
		if v, set := cfg.LiteLLM["redis"]; set {
			if redisMap, ok := v.(map[string]interface{}); ok {
				if enabled, ok := redisMap["enabled"].(bool); ok && enabled {
					warnings = append(warnings, "litellm.redis.enabled: true will be overridden to false — InferenceHub provides a dedicated Redis for LiteLLM (redis.litellm) wired via the litellm-env secret")
				}
			}
		}

		// Warn if model_list is set directly (should come from models: section)
		if pc, ok := cfg.LiteLLM["proxy_config"].(map[string]interface{}); ok {
			if _, set := pc["model_list"]; set {
				warnings = append(warnings, "litellm.proxy_config.model_list is generated from the models: section — remove it from inferencehub.yaml")
			}
		}
	}

	return nil, warnings
}

// isValidK8sName checks if a string is a valid Kubernetes resource name.
func isValidK8sName(name string) bool {
	matched, _ := regexp.MatchString(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`, name)
	return matched && len(name) <= 253
}

// isValidDomain checks if a string is a valid FQDN.
func isValidDomain(domain string) bool {
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9][a-zA-Z0-9\-_.]*\.[a-zA-Z]{2,}$`, domain)
	return matched
}
