package config

// Config represents the inferencehub.yaml user configuration.
// All secrets must be provided via environment variables (see .env.example).
// Use ${VAR_NAME} syntax in config.yaml to reference environment variables.
type Config struct {
	// ClusterName is the Kubernetes cluster name (used by prerequisites script)
	ClusterName string `yaml:"clusterName" json:"clusterName"`

	// Domain is the fully qualified domain name for this InferenceHub instance
	Domain string `yaml:"domain" json:"domain"`

	// Environment controls TLS issuer selection:
	//   production|prod  → letsencrypt-prod
	//   all others       → letsencrypt-staging
	Environment string `yaml:"environment" json:"environment"`

	// Namespace for InferenceHub resources (default: inferencehub)
	Namespace string `yaml:"namespace,omitempty" json:"namespace,omitempty"`

	// CloudProvider identifies the cloud environment for auto-selecting Helm values.
	// Supported: aws, gcp, azure, local (empty = no provider-specific overrides).
	// Maps to helm/inferencehub/values-{provider}.yaml within the chart directory.
	// Can be overridden at runtime with the --cloud-provider CLI flag.
	CloudProvider string `yaml:"cloudProvider,omitempty" json:"cloudProvider,omitempty"`

	// ChartPath is an optional override to the Helm chart path
	ChartPath string `yaml:"chartPath,omitempty" json:"chartPath,omitempty"`

	// KubeconfigPath is an optional override to the kubeconfig file
	KubeconfigPath string `yaml:"kubeconfigPath,omitempty" json:"kubeconfigPath,omitempty"`

	// Gateway references the existing Gateway resource created by prerequisites script
	Gateway GatewayConfig `yaml:"gateway" json:"gateway"`

	// Versions allows pinning or upgrading component image versions
	Versions VersionConfig `yaml:"versions,omitempty" json:"versions,omitempty"`

	// Models configures LLM models grouped by provider
	Models ModelsConfig `yaml:"models" json:"models"`

	// PostgreSQL configures the database; empty URL means use in-cluster container
	PostgreSQL DatastoreConfig `yaml:"postgresql,omitempty" json:"postgresql,omitempty"`

	// Redis configures separate caches for OpenWebUI and LiteLLM.
	// Each app gets its own Redis to avoid eviction policy conflicts.
	// Empty URL means use in-cluster container for that app.
	Redis RedisConfig `yaml:"redis,omitempty" json:"redis,omitempty"`

	// Observability configures optional LLM observability (Langfuse)
	Observability ObservabilityConfig `yaml:"observability,omitempty" json:"observability,omitempty"`

	// AWS holds AWS-specific settings used when cloudProvider is "aws".
	AWS AWSConfig `yaml:"aws,omitempty" json:"aws,omitempty"`

	// OpenWebUI is a raw passthrough to the open-webui subchart values.
	// Values are merged with InferenceHub's required injections.
	// Always overridden (do not set): openaiBaseApiUrl, ollama.enabled,
	// websocket.url (computed from redis: config), websocket.redis.enabled,
	// extraEnvVars entries for DATABASE_URL and OPENAI_API_KEY.
	// websocket.redis.* sub-fields (image, resources, labels) are ignored — InferenceHub's Redis is used.
	// Safe to set: websocket.enabled, websocket.nodeSelector, pipelines.enabled, and all other keys.
	// Reference: https://github.com/open-webui/helm-charts/blob/main/charts/open-webui/values.yaml
	OpenWebUI map[string]interface{} `yaml:"openwebui,omitempty" json:"openwebui,omitempty"`

	// LiteLLM is a raw passthrough to the litellm-helm subchart values.
	// Values are merged with InferenceHub's required injections.
	// Protected keys (masterkeySecretName, db.deployStandalone, environmentSecrets entry for
	// inferencehub-litellm-env, and proxy_config.model_list) are always overridden.
	// Reference: https://github.com/BerriAI/litellm/blob/main/deploy/charts/litellm-helm/values.yaml
	LiteLLM map[string]interface{} `yaml:"litellm,omitempty" json:"litellm,omitempty"`
}

// GatewayConfig references the Gateway resource managed by the prerequisites script.
// Run scripts/setup-prerequisites.py to create this gateway.
type GatewayConfig struct {
	Name      string `yaml:"name" json:"name"`
	Namespace string `yaml:"namespace" json:"namespace"`
}

// VersionConfig allows overriding component image tags.
// These override the chart defaults and allow upgrade/downgrade from config.yaml.
type VersionConfig struct {
	OpenWebUI  string `yaml:"openwebui,omitempty" json:"openwebui,omitempty"`
	LiteLLM    string `yaml:"litellm,omitempty" json:"litellm,omitempty"`
	PostgreSQL string `yaml:"postgresql,omitempty" json:"postgresql,omitempty"`
	Redis      string `yaml:"redis,omitempty" json:"redis,omitempty"`
}

// ModelsConfig holds model definitions grouped by provider.
// Note: self-hosted models like Ollama require no extra configuration beyond apiBase.
type ModelsConfig struct {
	Bedrock []BedrockModel `yaml:"bedrock,omitempty" json:"bedrock,omitempty"`
	OpenAI  []OpenAIModel  `yaml:"openai,omitempty" json:"openai,omitempty"`
	Ollama  []OllamaModel  `yaml:"ollama,omitempty" json:"ollama,omitempty"`
	Azure   []AzureModel   `yaml:"azure,omitempty" json:"azure,omitempty"`
}

// TotalCount returns the total number of configured models.
func (m *ModelsConfig) TotalCount() int {
	return len(m.Bedrock) + len(m.OpenAI) + len(m.Ollama) + len(m.Azure)
}

// HasModels returns true if at least one model is configured.
func (m *ModelsConfig) HasModels() bool {
	return m.TotalCount() > 0
}

// BedrockModel configures an AWS Bedrock model.
// Requires an IRSA role with bedrock:InvokeModel permissions.
type BedrockModel struct {
	Name   string `yaml:"name" json:"name"`
	Model  string `yaml:"model" json:"model"`
	Region string `yaml:"region" json:"region"`
}

// OpenAIModel configures an OpenAI API model.
// Requires OPENAI_API_KEY environment variable.
type OpenAIModel struct {
	Name  string `yaml:"name" json:"name"`
	Model string `yaml:"model" json:"model"`
}

// OllamaModel configures a self-hosted Ollama model.
// No extra infrastructure required; just run Ollama on any accessible host.
type OllamaModel struct {
	Name    string `yaml:"name" json:"name"`
	Model   string `yaml:"model" json:"model"`
	APIBase string `yaml:"apiBase" json:"apiBase"`
}

// AzureModel configures an Azure OpenAI deployment.
type AzureModel struct {
	Name       string `yaml:"name" json:"name"`
	Model      string `yaml:"model" json:"model"`
	APIBase    string `yaml:"apiBase" json:"apiBase"`
	APIVersion string `yaml:"apiVersion,omitempty" json:"apiVersion,omitempty"`
}

// AWSConfig holds AWS-specific deployment settings.
// These are applied as programmatic Helm overrides so users never need to
// edit values-aws.yaml directly.
type AWSConfig struct {
	// LiteLLMRoleARN is the IAM Role ARN annotated onto the LiteLLM service account
	// (eks.amazonaws.com/role-arn) so that LiteLLM pods can call AWS Bedrock via IRSA.
	// Required when any models.bedrock entries are configured.
	// Format: arn:aws:iam::<account-id>:role/<role-name>
	LiteLLMRoleARN string `yaml:"litellmRoleArn,omitempty" json:"litellmRoleArn,omitempty"`
}

// DatastoreConfig configures an optional external datastore.
// If URL is empty, the in-cluster container defined in values.yaml is used.
// Use ${VAR} syntax to reference environment variables for passwords.
type DatastoreConfig struct {
	URL      string `yaml:"url,omitempty" json:"url,omitempty"`
	Username string `yaml:"username,omitempty" json:"username,omitempty"`
	// Password supports ${ENV_VAR} syntax for environment variable interpolation
	Password string `yaml:"password,omitempty" json:"password,omitempty"`

	// For external PostgreSQL only: per-app connection strings.
	// If set, these take precedence over URL/Username/Password.
	OpenWebUIConnectionString string `yaml:"openwebuiConnectionString,omitempty" json:"openwebuiConnectionString,omitempty"`
	LiteLLMConnectionString   string `yaml:"litellmConnectionString,omitempty" json:"litellmConnectionString,omitempty"`
}

// IsExternal returns true when an external datastore URL is provided.
func (d *DatastoreConfig) IsExternal() bool {
	return d.URL != "" || d.OpenWebUIConnectionString != "" || d.LiteLLMConnectionString != ""
}

// RedisConfig holds per-app Redis connection settings.
// InferenceHub deploys separate Redis instances for OpenWebUI and LiteLLM
// to avoid eviction policy conflicts (session state vs. API caching).
type RedisConfig struct {
	// OpenWebUI configures the Redis used by OpenWebUI for websocket session state.
	// Recommended eviction policy: noeviction or volatile-lru.
	OpenWebUI RedisAppConfig `yaml:"openwebui,omitempty" json:"openwebui,omitempty"`

	// LiteLLM configures the Redis used by LiteLLM for API response caching.
	// Recommended eviction policy: allkeys-lru.
	LiteLLM RedisAppConfig `yaml:"litellm,omitempty" json:"litellm,omitempty"`
}

// RedisAppConfig configures a single Redis instance for one application.
// If URL is empty, the in-cluster container is used.
// Use ${VAR} syntax to reference environment variables for passwords.
type RedisAppConfig struct {
	// URL is the full Redis connection URL (e.g., redis://host:6379).
	// Set to use an external Redis; leave empty for in-cluster.
	URL string `yaml:"url,omitempty" json:"url,omitempty"`
	// Password supports ${ENV_VAR} syntax for environment variable interpolation.
	Password string `yaml:"password,omitempty" json:"password,omitempty"`
}

// IsExternal returns true when an external Redis URL is provided.
func (r *RedisAppConfig) IsExternal() bool {
	return r.URL != ""
}

// ObservabilityConfig configures optional LLM observability.
// Disabled by default. Enable to send traces to Langfuse.
type ObservabilityConfig struct {
	Enabled  bool          `yaml:"enabled" json:"enabled"`
	Langfuse LangfuseConfig `yaml:"langfuse,omitempty" json:"langfuse,omitempty"`
}

// LangfuseConfig configures Langfuse LLM observability.
// Use ${VAR} syntax for publicKey and secretKey.
type LangfuseConfig struct {
	Host      string `yaml:"host,omitempty" json:"host,omitempty"`
	PublicKey string `yaml:"publicKey,omitempty" json:"publicKey,omitempty"`
	SecretKey string `yaml:"secretKey,omitempty" json:"secretKey,omitempty"`
}

// IssuerType derives the Let's Encrypt issuer name from the environment.
// production|prod → letsencrypt-prod; everything else → letsencrypt-staging
func (c *Config) IssuerType() string {
	switch c.Environment {
	case "production", "prod":
		return "letsencrypt-prod"
	default:
		return "letsencrypt-staging"
	}
}

// EffectiveNamespace returns the configured namespace or the default "inferencehub".
func (c *Config) EffectiveNamespace() string {
	if c.Namespace != "" {
		return c.Namespace
	}
	return "inferencehub"
}
