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

	// Validate external Redis config
	if cfg.Redis.IsExternal() {
		if cfg.Redis.Password == "" {
			return fmt.Errorf("redis.password is required when redis.url is set")
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

	if !cfg.Redis.IsExternal() {
		warnings = append(warnings, "Using in-cluster Redis. For production, consider an external managed cache (e.g., ElastiCache)")
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
