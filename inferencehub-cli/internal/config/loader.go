package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/joho/godotenv"
	"sigs.k8s.io/yaml"
)

// envVarPattern matches ${VAR_NAME} placeholders in config values.
var envVarPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

// Load reads config.yaml from the given path, resolves ${ENV_VAR} placeholders,
// and returns the parsed Config. Secrets must be set as environment variables.
//
// Environment variable loading order (later overrides earlier):
//  1. ~/.inferencehub/.env
//  2. ./.env
//  3. ./.env.local
func Load(configPath string) (*Config, error) {
	loadEnvFiles()

	if configPath == "" {
		return nil, fmt.Errorf("config file path is required (use --config)")
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", configPath)
	}

	raw, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	// Interpolate ${VAR} placeholders before YAML parsing
	interpolated := interpolateEnvVars(string(raw))

	var cfg Config
	if err := yaml.Unmarshal([]byte(interpolated), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", configPath, err)
	}

	// Apply defaults for optional fields
	applyDefaults(&cfg)

	return &cfg, nil
}

// interpolateEnvVars replaces all ${VAR_NAME} occurrences with the corresponding
// environment variable value. Unset variables are replaced with an empty string.
func interpolateEnvVars(s string) string {
	return envVarPattern.ReplaceAllStringFunc(s, func(match string) string {
		// Extract variable name from ${VAR_NAME}
		sub := envVarPattern.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		varName := sub[1]
		if val := os.Getenv(varName); val != "" {
			return val
		}
		// Return empty string if env var not set (validator will catch missing required vars)
		return ""
	})
}

// applyDefaults sets sensible defaults for optional fields.
func applyDefaults(cfg *Config) {
	if cfg.Namespace == "" {
		cfg.Namespace = "inferencehub"
	}
	if cfg.Gateway.Name == "" {
		cfg.Gateway.Name = "inferencehub-gateway"
	}
	if cfg.Gateway.Namespace == "" {
		cfg.Gateway.Namespace = "envoy-gateway-system"
	}
	if cfg.Observability.Langfuse.Host == "" && cfg.Observability.Enabled {
		cfg.Observability.Langfuse.Host = "https://cloud.langfuse.com"
	}
}

// loadEnvFiles loads .env files from standard locations in order of precedence.
// Later files override earlier ones. Errors are silently ignored.
func loadEnvFiles() {
	envFiles := envFilePaths()
	for _, f := range envFiles {
		if _, err := os.Stat(f); err == nil {
			_ = godotenv.Load(f)
		}
	}
}

// GetLoadedEnvFiles returns the list of .env files that exist and were (or would be) loaded.
func GetLoadedEnvFiles() []string {
	var loaded []string
	for _, f := range envFilePaths() {
		if _, err := os.Stat(f); err == nil {
			loaded = append(loaded, f)
		}
	}
	return loaded
}

// envFilePaths returns the ordered list of .env file paths to check.
func envFilePaths() []string {
	var paths []string

	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".inferencehub", ".env"))
	}

	cwd, err := os.Getwd()
	if err == nil {
		paths = append(paths,
			filepath.Join(cwd, ".env"),
			filepath.Join(cwd, ".env.local"),
		)
	}

	return paths
}

// SaveToFile marshals cfg to YAML and writes it to the given path.
func SaveToFile(cfg *Config, path string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}
