# Configuration Reference

InferenceHub is configured via a single YAML file. The CLI interpolates `${ENV_VAR}` placeholders before parsing, so secrets never have to be written in plain text.

## Generate a starter config

```bash
inferencehub config init
# Creates inferencehub.yaml in the current directory
```

## Full schema

```yaml
# --- Cluster identity ---
clusterName: my-cluster          # Used for labelling, required
domain: inferencehub.ai          # Hostname where OpenWebUI is served, required
environment: production          # Affects TLS issuer: "production"/"prod" → letsencrypt-prod,
                                 # anything else → letsencrypt-staging
namespace: inferencehub          # Kubernetes namespace (default: inferencehub)

# --- Gateway API ---
gateway:
  name: inferencehub-gateway          # Name of the Gateway resource
  namespace: envoy-gateway-system     # Namespace where the Gateway lives

# --- Component versions (optional) ---
# Defaults match the chart's pinned versions. Override only when needed.
versions:
  openwebui: "v0.8.5"
  litellm: "main-v1.81.12-stable.2"
  postgresql: "18-alpine"
  redis: "8-alpine"
  searxng: "2026.3.6-0716de6bc"

# --- Models ---
models:
  bedrock:
    - name: claude-sonnet           # Display name in OpenWebUI
      model: anthropic.claude-3-5-sonnet-20241022-v2:0
      region: us-east-1             # AWS region, required for Bedrock
  openai:
    - name: gpt-4o
      model: gpt-4o
  ollama:
    - name: llama3
      model: llama3.2:3b
      apiBase: http://ollama.default.svc.cluster.local:11434
  azure:
    - name: gpt-4-azure
      model: gpt-4
      apiBase: https://YOUR_RESOURCE.openai.azure.com
      apiVersion: "2024-02-01"

# --- Storage class (optional) ---
# Applied to all in-cluster PVCs: OpenWebUI, PostgreSQL, both Redis instances.
# Leave unset to auto-detect the cluster's annotated default StorageClass.
# Set explicitly when no StorageClass is annotated as default (e.g. EKS with gp2):
storageClass: gp2

# --- Cloud provider ---
# Selects helm/inferencehub/values-{provider}.yaml automatically.
cloudProvider: aws     # aws | gcp | azure | local

# --- AWS settings (used when cloudProvider: aws) ---
aws:
  litellmRoleArn: "arn:aws:iam::123456789012:role/litellm-bedrock-role"

# --- External datastores (optional) ---
# Leave blank to use in-cluster PostgreSQL and Redis pods.

# External PostgreSQL — provide per-app connection strings (v0.2.0+)
postgresql:
  openwebuiConnectionString: "postgresql://user:${POSTGRES_PASSWORD}@mydb.us-east-1.rds.amazonaws.com:5432/openwebui"
  litellmConnectionString:   "postgresql://user:${POSTGRES_PASSWORD}@mydb.us-east-1.rds.amazonaws.com:5432/litellm"

# External Redis — configure per app (each has its own in-cluster pod by default)
# See docs/infrastructure.md for why Redis is split.
redis:
  openwebui:
    url: "redis://openwebui-cache.abc123.cache.amazonaws.com:6379"
    password: "${OPENWEBUI_REDIS_PASSWORD}"
  litellm:
    url: "redis://litellm-cache.def456.cache.amazonaws.com:6379"
    password: "${LITELLM_REDIS_PASSWORD}"

# --- Web search (optional) ---
# Deploys SearXNG in-cluster by default when enabled: true.
# To use an external engine, set external.enabled: true.
webSearch:
  enabled: false
  engine: searxng     # searxng (default) | brave | bing | tavily | google_pse | duckduckgo
  external:
    enabled: false
    queryUrl: ""      # for external searxng: https://searxng.example.com/search?q=<query>&format=json
    apiKey: "${SEARCH_API_KEY}"   # for brave / bing / tavily / google_pse
    engineId: ""      # for google_pse only

# --- Observability ---
observability:
  enabled: false
  langfuse:
    host: https://cloud.langfuse.com
    publicKey: "${LANGFUSE_PUBLIC_KEY}"
    secretKey: "${LANGFUSE_SECRET_KEY}"

# --- Passthrough: OpenWebUI subchart values ---
# Any key accepted by the open-webui Helm chart can be set here.
# InferenceHub merges its required injections on top — see "Protected keys" below.
# Full upstream reference: https://github.com/open-webui/helm-charts/blob/main/charts/open-webui/values.yaml
openwebui:
  # Example: require admin approval before new users can log in
  defaultUserRole: pending
  # Example: enable SSO via OIDC
  # sso:
  #   enabled: true
  #   oidc:
  #     enabled: true
  #     clientId: "${OIDC_CLIENT_ID}"
  #     clientSecret: "${OIDC_CLIENT_SECRET}"
  #     providerUrl: "https://accounts.google.com"
  #     providerName: "Google"
  # Example: enable the Pipelines feature
  # pipelines:
  #   enabled: true
  # Example: override resource limits
  # resources:
  #   limits:
  #     cpu: "2"
  #     memory: 4Gi

# --- Passthrough: LiteLLM subchart values ---
# Any key accepted by the litellm-helm chart can be set here.
# InferenceHub merges its required injections on top — see "Protected keys" below.
# Full upstream reference: https://github.com/BerriAI/litellm/blob/main/deploy/charts/litellm-helm/values.yaml
litellm:
  # Example: tune proxy settings
  # proxy_config:
  #   general_settings:
  #     request_timeout: 600
  #   litellm_settings:
  #     drop_params: true
  # Example: configure Slack alerting
  # proxy_config:
  #   general_settings:
  #     alerting:
  #       - slack
  #     slack_msg_webhook_url: "${SLACK_WEBHOOK_URL}"
```

---

## Value precedence

InferenceHub merges configuration from multiple sources before deploying. When the same key is defined in multiple places, the highest priority source wins.

**Priority (highest to lowest):**

1.  **Component-level passthrough** (`openwebui:`, `litellm:`, etc.) in `inferencehub.yaml`.
2.  **Global fields** (`storageClass:`, `versions:`, etc.) in `inferencehub.yaml`.
3.  **Cloud provider presets** (e.g., `helm/inferencehub/values-aws.yaml`), auto-selected via `cloudProvider:`.
4.  **Cluster auto-detection** (e.g., detecting the default StorageClass from the cluster API).
5.  **Chart defaults** (`helm/inferencehub/values.yaml`).

---

## Field reference

### Top-level fields

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `clusterName` | Yes | — | Cluster identifier used in labels |
| `domain` | Yes | — | Hostname for OpenWebUI (no scheme) |
| `environment` | Yes | — | `production`/`prod` = Let's Encrypt prod certs; anything else = staging |
| `namespace` | No | `inferencehub` | Kubernetes namespace |
| `cloudProvider` | No | — | `aws`, `gcp`, `azure`, `local` — auto-selects `values-{provider}.yaml` |
| `storageClass` | No | — | StorageClass for all in-cluster PVCs — see [`storageClass`](#storageclass) |
| `webSearch.enabled` | No | `false` | Enable web search in OpenWebUI — see [`webSearch`](#websearch) |

### `gateway`

| Field | Default | Description |
|-------|---------|-------------|
| `name` | `inferencehub-gateway` | Name of the Gateway resource created by the prerequisites script |
| `namespace` | `envoy-gateway-system` | Namespace where that Gateway lives |

The HTTPRoute created by the Helm chart attaches to this gateway. Must match what the prerequisites script created.

### `models`

Models are grouped by provider. At least one model is required. The CLI translates this section into LiteLLM's `proxy_config.model_list` automatically.

#### `models.bedrock[]`

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Display name shown in OpenWebUI |
| `model` | Yes | Full Bedrock model ID |
| `region` | Yes | AWS region |

LiteLLM calls Bedrock via IRSA. Set `aws.litellmRoleArn` — the CLI annotates the LiteLLM service account automatically. See [`aws`](#aws) below.

#### `models.openai[]`

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Display name |
| `model` | Yes | OpenAI model ID (e.g., `gpt-4o`) |

Set `OPENAI_API_KEY` in your `.env` file. The CLI injects it into OpenWebUI and LiteLLM via secrets.

#### `models.ollama[]`

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Display name |
| `model` | Yes | Ollama model ID (e.g., `llama3.2:3b`) |
| `apiBase` | Yes | URL of the Ollama server (external to the cluster) |

#### `models.azure[]`

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Display name |
| `model` | Yes | Azure deployment name |
| `apiBase` | Yes | Azure OpenAI endpoint URL |
| `apiVersion` | No | API version (e.g., `2024-02-01`) |

Set `AZURE_OPENAI_API_KEY` in your `.env` file.

### `postgresql`

Leave all fields blank to use the in-cluster PostgreSQL pod (default). To use an external database, provide per-app connection strings:

| Field | Description |
|-------|-------------|
| `openwebuiConnectionString` | Full connection string for the OpenWebUI database |
| `litellmConnectionString` | Full connection string for the LiteLLM database |

```yaml
postgresql:
  openwebuiConnectionString: "postgresql://user:${POSTGRES_PASSWORD}@mydb.us-east-1.rds.amazonaws.com:5432/openwebui"
  litellmConnectionString:   "postgresql://user:${POSTGRES_PASSWORD}@mydb.us-east-1.rds.amazonaws.com:5432/litellm"
```

When either connection string is set, the in-cluster PostgreSQL pod is disabled automatically. Both strings must be provided together.

### `redis`

InferenceHub deploys separate Redis instances for OpenWebUI (session state) and LiteLLM (API caching) to avoid eviction policy conflicts. See [docs/infrastructure.md](infrastructure.md) for details.

Each app has its own sub-block:

| Field | Description |
|-------|-------------|
| `redis.openwebui.url` | External Redis URL for OpenWebUI (leave blank for in-cluster) |
| `redis.openwebui.password` | Redis auth password (or use `${OPENWEBUI_REDIS_PASSWORD}`) |
| `redis.litellm.url` | External Redis URL for LiteLLM (leave blank for in-cluster) |
| `redis.litellm.password` | Redis auth password (or use `${LITELLM_REDIS_PASSWORD}`) |

You can mix in-cluster and external — configure only the app that needs an external Redis:

```yaml
redis:
  litellm:
    url: "redis://litellm-cache.def456.cache.amazonaws.com:6379"
    password: "${LITELLM_REDIS_PASSWORD}"
  # openwebui uses in-cluster Redis (default — omit to use in-cluster)
```

### `webSearch`

Enables web search inside OpenWebUI. When `enabled: true` and `external.enabled: false`, InferenceHub deploys a [SearXNG](https://github.com/searxng/searxng) pod in-cluster and wires it to OpenWebUI automatically.

#### In-cluster SearXNG (default)

```yaml
webSearch:
  enabled: true
```

**Authentication (`SEARXNG_SECRET_KEY`):** <br/>
- **Automatic:** The CLI generates a random 32-character key during every install or upgrade if not provided. <br/>
- **Manual:** To use a specific key, set the `SEARXNG_SECRET_KEY` environment variable in your `.env` file or shell.

#### External SearXNG instance

```yaml
webSearch:
  enabled: true
  external:
    enabled: true
    queryUrl: "https://searxng.example.com/search?q=<query>&format=json"
```

No in-cluster SearXNG is deployed. The provided `queryUrl` is injected into OpenWebUI directly.

#### External API-key engine

Set `engine` to any OpenWebUI-supported provider and supply credentials via `apiKey`:

```yaml
webSearch:
  enabled: true
  engine: brave
  external:
    enabled: true
    apiKey: "${BRAVE_API_KEY}"
```

| `engine` value | Provider | Required fields |
|----------------|----------|-----------------|
| `searxng` | Self-hosted SearXNG | `queryUrl` |
| `brave` | Brave Search | `apiKey` |
| `bing` | Bing Web Search | `apiKey` |
| `tavily` | Tavily | `apiKey` |
| `google_pse` | Google Programmable Search Engine | `apiKey`, `engineId` |
| `duckduckgo` | DuckDuckGo | _(none)_ |

#### `webSearch` field reference

| Field | Default | Description |
|-------|---------|-------------|
| `enabled` | `false` | Enable web search in OpenWebUI |
| `engine` | `searxng` | Search backend — see table above |
| `external.enabled` | `false` | Use a user-supplied engine instead of deploying in-cluster SearXNG |
| `external.queryUrl` | — | Full query URL with `<query>` placeholder (SearXNG only) |
| `external.apiKey` | — | API key for the search engine. Supports `${ENV_VAR}` syntax |
| `external.engineId` | — | Search engine ID (Google PSE only) |

### `observability`

| Field | Default | Description |
|-------|---------|-------------|
| `enabled` | `false` | Enable Langfuse integration |
| `langfuse.host` | `https://cloud.langfuse.com` | Langfuse server URL |
| `langfuse.publicKey` | — | Langfuse public key |
| `langfuse.secretKey` | — | Langfuse secret key |

When `enabled: true`, all LiteLLM requests are traced in Langfuse.

### `aws`

Settings used when `cloudProvider: aws`.

| Field | Required | Description |
|-------|----------|-------------|
| `litellmRoleArn` | Yes (if using Bedrock) | IAM Role ARN for the LiteLLM service account. Annotated as `eks.amazonaws.com/role-arn` so LiteLLM pods assume this role via IRSA. Format: `arn:aws:iam::<account-id>:role/<name>` |

The CLI annotates the LiteLLM service account with this ARN automatically during `install` and `upgrade`. Without it, LiteLLM falls back to the node instance role, which typically lacks `bedrock:InvokeModel` permissions.

### `storageClass`

Sets the Kubernetes StorageClass for all in-cluster PVCs: OpenWebUI data, PostgreSQL, and both Redis instances.

```yaml
storageClass: gp2
```

**Selection priority (highest wins):**

| Priority | Source | Example |
|----------|--------|---------|
| 1 — most specific | Component-level value in `openwebui:`/`litellm:` passthrough | `openwebui.persistence.storageClass: gp3` |
| 2 | Global `storageClass:` in `inferencehub.yaml` | `storageClass: gp2` |
| 3 | Cloud provider preset (e.g., `values-aws.yaml`) | `storageClass: gp3` |
| 4 | Cluster default annotation (`storageclass.kubernetes.io/is-default-class: "true"`) | Auto-detected |
| 5 — fallback | Nothing set | PVC remains pending |


**When to set this field:**

- Your cluster has no StorageClass annotated as default (common on EKS — `gp2` exists but is not marked default unless you annotate it)
- You want to pin a specific class for all components regardless of cluster defaults

**When to leave it unset:**

- Your cluster has a default StorageClass annotation — InferenceHub detects and uses it automatically

**Overriding per-component** — if you need different storage classes for different components, use the `openwebui:` passthrough and skip `storageClass:` entirely:

```yaml
openwebui:
  persistence:
    storageClass: gp3   # SSD for OpenWebUI data

# postgresql and redis will still use cluster default or remain unset
```

---

### `openwebui` — passthrough

The `openwebui:` key is a raw passthrough to the [open-webui Helm chart](https://github.com/open-webui/helm-charts/blob/main/charts/open-webui/values.yaml). Any value the upstream chart accepts can be set here. InferenceHub deep-merges its required injections on top of your values.

**How `extraEnvVars` merging works (three tiers):**

The CLI applies a three-tier merge when building `openwebui.extraEnvVars`:

1. **Soft defaults** — set by the CLI based on your `webSearch:` config (lowest priority). Your `openwebui.extraEnvVars` entries with the same name **override** these.
2. **User values** — whatever you supply under `openwebui.extraEnvVars`.
3. **Truly protected** — always injected last and always win, regardless of what you set.

**Truly protected keys** — always set by InferenceHub, always override user values:

| Key | Managed value |
|-----|---------------|
| `openaiBaseApiUrl` | Wired to the LiteLLM service |
| `extraEnvVars[DATABASE_URL]` | Injected from the PostgreSQL secret |
| `extraEnvVars[OPENAI_API_KEY]` | Injected from the LiteLLM master key secret |
| `ollama.enabled` | Always `false` — use `models.ollama` with an external URL |
| `websocket.redis.enabled` | Always `false` — InferenceHub provides a dedicated Redis for OpenWebUI (`redis.openwebui`) |
| `websocket.url` | Computed from `redis.openwebui` config |

**CLI soft defaults** — set automatically when `webSearch.enabled: true`, but can be overridden via `openwebui.extraEnvVars`:

| Key | Default value |
|-----|---------------|
| `extraEnvVars[ENABLE_RAG_WEB_SEARCH]` | `"true"` (from `webSearch.enabled`) |
| `extraEnvVars[RAG_WEB_SEARCH_ENGINE]` | Value of `webSearch.engine` (default: `searxng`) |
| `extraEnvVars[SEARXNG_QUERY_URL]` | In-cluster SearXNG URL |
| `extraEnvVars[BRAVE_SEARCH_API_KEY]` | Value of `webSearch.external.apiKey` |
| `extraEnvVars[BING_SEARCH_V7_SUBSCRIPTION_KEY]` | Value of `webSearch.external.apiKey` |
| `extraEnvVars[TAVILY_API_KEY]` | Value of `webSearch.external.apiKey` |
| `extraEnvVars[GOOGLE_PSE_API_KEY]` | Value of `webSearch.external.apiKey` |

> **Note:** For web search configuration, always use the `webSearch:` block. Setting search engine env vars directly in `openwebui.extraEnvVars` will configure OpenWebUI but will **not** deploy an in-cluster SearXNG pod.

**Safe to set (examples):**

```yaml
openwebui:
  defaultUserRole: pending        # require admin approval for new signups

  pipelines:
    enabled: true                 # enable the Pipelines feature

  sso:
    enabled: true
    oidc:
      enabled: true
      clientId: "${OIDC_CLIENT_ID}"
      clientSecret: "${OIDC_CLIENT_SECRET}"
      providerUrl: "https://accounts.google.com"
      providerName: "Google"

  resources:
    limits:
      cpu: "2"
      memory: 4Gi

  websocket:
    enabled: false                # disable websockets entirely (not recommended)

  persistence:
    storageClass: gp3             # override StorageClass for OpenWebUI PVC only
                                  # (takes precedence over top-level storageClass:)
    size: 10Gi                    # override PVC size (default: 2Gi)
```

### `litellm` — passthrough

The `litellm:` key is a raw passthrough to the [litellm-helm chart](https://github.com/BerriAI/litellm/blob/main/deploy/charts/litellm-helm/values.yaml). Any value the upstream chart accepts can be set here. InferenceHub deep-merges its required injections on top.

**Protected keys** — always overridden by InferenceHub:

| Key | Managed value |
|-----|---------------|
| `masterkeySecretName` / `masterkeySecretKey` | Points to InferenceHub's managed secret |
| `db.deployStandalone` | Always `false` — InferenceHub provides PostgreSQL |
| `db.useExisting` | Always `true` |
| `redis.enabled` | Always `false` — InferenceHub provides LiteLLM Redis |
| `environmentSecrets` | InferenceHub appends its wiring secret (`DATABASE_URL`, `REDIS_HOST`, etc.) |
| `proxy_config.model_list` | Generated from the `models:` section |
| `proxy_config.general_settings.master_key` | Set from `LITELLM_MASTER_KEY` |

**Safe to set (examples):**

```yaml
litellm:
  proxy_config:
    general_settings:
      request_timeout: 600        # increase timeout for long-running model calls
      alerting:
        - slack
      slack_msg_webhook_url: "${SLACK_WEBHOOK_URL}"
    litellm_settings:
      drop_params: true           # ignore unsupported params instead of erroring
      success_callback:
        - langfuse

  resources:
    limits:
      cpu: "2"
      memory: 2Gi

  replicaCount: 2                 # scale LiteLLM horizontally
```

---

## Environment variables

The CLI reads env vars in two ways:

1. **Auto-loaded `.env` files** (applied in order):
   - `~/.inferencehub/.env`
   - `./.env`
   - `./.env.local`

2. **Interpolation** — any `${VAR_NAME}` in the config YAML is replaced with the environment variable value before parsing.

### Required

| Variable | Description |
|----------|-------------|
| `LITELLM_MASTER_KEY` | API key for the LiteLLM gateway. Must start with `sk-`. |
| `POSTGRES_PASSWORD` | Required when using in-cluster PostgreSQL |

### Optional

| Variable | Description |
|----------|-------------|
| `OPENWEBUI_REDIS_PASSWORD` | Auth password for the OpenWebUI Redis (in-cluster or external) |
| `LITELLM_REDIS_PASSWORD` | Auth password for the LiteLLM Redis (in-cluster or external) |
| `LANGFUSE_PUBLIC_KEY` | Langfuse public key (when `observability.enabled: true`) |
| `LANGFUSE_SECRET_KEY` | Langfuse secret key (when `observability.enabled: true`) |
| `OPENAI_API_KEY` | OpenAI key (when using `models.openai`) |
| `AZURE_OPENAI_API_KEY` | Azure key (when using `models.azure`) |
| `BRAVE_API_KEY` | Brave Search API key (when `webSearch.engine: brave`) |
| `BING_API_KEY` | Bing API key (when `webSearch.engine: bing`) |
| `TAVILY_API_KEY` | Tavily API key (when `webSearch.engine: tavily`) |
| `GOOGLE_PSE_API_KEY` | Google PSE API key (when `webSearch.engine: google_pse`) |

---

## Example: AWS Bedrock with SSO and external datastores

```yaml
clusterName: prod-eks
domain: inferencehub.ai
environment: production
namespace: inferencehub
cloudProvider: aws

gateway:
  name: inferencehub-gateway
  namespace: envoy-gateway-system

models:
  bedrock:
    - name: claude-sonnet
      model: anthropic.claude-3-5-sonnet-20241022-v2:0
      region: us-east-1

aws:
  litellmRoleArn: "arn:aws:iam::123456789012:role/litellm-bedrock-role"

postgresql:
  openwebuiConnectionString: "postgresql://inferencehub:${POSTGRES_PASSWORD}@mydb.us-east-1.rds.amazonaws.com:5432/openwebui"
  litellmConnectionString:   "postgresql://inferencehub:${POSTGRES_PASSWORD}@mydb.us-east-1.rds.amazonaws.com:5432/litellm"

redis:
  openwebui:
    url: "redis://openwebui-cache.abc123.cache.amazonaws.com:6379"
    password: "${OPENWEBUI_REDIS_PASSWORD}"
  litellm:
    url: "redis://litellm-cache.def456.cache.amazonaws.com:6379"
    password: "${LITELLM_REDIS_PASSWORD}"

observability:
  enabled: true
  langfuse:
    host: https://cloud.langfuse.com
    publicKey: "${LANGFUSE_PUBLIC_KEY}"
    secretKey: "${LANGFUSE_SECRET_KEY}"

webSearch:
  enabled: true         # deploys SearXNG in-cluster automatically

openwebui:
  defaultUserRole: pending
  sso:
    enabled: true
    oidc:
      enabled: true
      clientId: "${OIDC_CLIENT_ID}"
      clientSecret: "${OIDC_CLIENT_SECRET}"
      providerUrl: "https://sso.company.com"
      providerName: "Company SSO"

litellm:
  proxy_config:
    general_settings:
      request_timeout: 600
```

## Example: Local development (kind + Ollama)

```yaml
clusterName: kind-local
domain: localhost
environment: staging
namespace: inferencehub
cloudProvider: local

gateway:
  name: inferencehub-gateway
  namespace: envoy-gateway-system

models:
  ollama:
    - name: llama3
      model: llama3.2:3b
      apiBase: http://host.docker.internal:11434
```

Install with:

```bash
export LITELLM_MASTER_KEY="sk-local-dev"
export POSTGRES_PASSWORD="localdev"

inferencehub install --config inferencehub.yaml
# values-local.yaml is loaded automatically
```

---

## Validating your config

```bash
inferencehub config validate --config inferencehub.yaml
```

This checks:
- Required fields are present
- `LITELLM_MASTER_KEY` is set and starts with `sk-`
- Model groups have required provider-specific fields
- External datastores have required connection fields

Warnings (non-fatal) are printed for:
- Using in-cluster PostgreSQL or Redis in production
- Staging TLS certificates
- Missing IRSA role ARN when Bedrock models are configured
- Setting protected passthrough keys that will be overridden

To see the fully-resolved config with env vars substituted:

```bash
inferencehub config show --config inferencehub.yaml
```
