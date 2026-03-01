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
domain: ai.example.com           # Hostname where OpenWebUI is served, required
environment: staging             # Affects TLS issuer: "production"/"prod" → letsencrypt-prod,
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

# --- Models ---
models:
  bedrock:
    - name: claude-sonnet           # Display name in OpenWebUI
      model: anthropic.claude-3-5-sonnet-20241022-v2:0
      region: us-east-1             # AWS region, required for Bedrock
  openai:
    - name: gpt-4o
      model: gpt-4o
      apiKey: "${OPENAI_API_KEY}"   # Use env var interpolation for secrets
  ollama:
    - name: llama3
      model: llama3.2:3b
      apiBase: http://ollama.default.svc.cluster.local:11434
  azure:
    - name: gpt-4-azure
      model: gpt-4
      apiBase: https://YOUR_RESOURCE.openai.azure.com
      apiVersion: "2024-02-01"
      apiKey: "${AZURE_OPENAI_API_KEY}"

# --- Cloud provider ---
# Selects helm/inferencehub/values-{provider}.yaml automatically.
# Provider-specific credentials go in the matching block below, not in the values file.
cloudProvider: aws     # aws | gcp | azure | local

# --- AWS settings (used when cloudProvider: aws) ---
aws:
  litellmRoleArn: "arn:aws:iam::123456789012:role/litellm-bedrock-role"

# --- External datastores (optional) ---
# Leave blank to use in-cluster PostgreSQL and Redis
postgresql:
  url: ""          # e.g., postgresql://user:pass@host:5432/db
  username: ""
  password: "${POSTGRES_PASSWORD}"

redis:
  url: ""          # e.g., redis://host:6379
  username: ""
  password: "${REDIS_PASSWORD}"

# --- Observability ---
observability:
  enabled: false
  langfuse:
    host: https://cloud.langfuse.com
    publicKey: "${LANGFUSE_PUBLIC_KEY}"
    secretKey: "${LANGFUSE_SECRET_KEY}"

```

## Field reference

### Top-level fields

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `clusterName` | Yes | — | Cluster identifier used in labels |
| `domain` | Yes | — | Hostname for OpenWebUI (no scheme) |
| `environment` | No | `staging` | `production`/`prod` = Let's Encrypt prod certs |
| `namespace` | No | `inferencehub` | Kubernetes namespace |
| `cloudProvider` | No | — | `aws`, `gcp`, `azure`, `local` — auto-selects `values-{provider}.yaml` |

### `gateway`

| Field | Default | Description |
|-------|---------|-------------|
| `name` | `inferencehub-gateway` | Name of the Gateway resource created by the prerequisites script |
| `namespace` | `envoy-gateway-system` | Namespace where that Gateway lives |

The HTTPRoute created by the Helm chart attaches to this gateway. Must match what the prerequisites script created.

### `models`

Models are grouped by provider. At least one model is required.

#### `models.bedrock[]`

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Display name shown in OpenWebUI |
| `model` | Yes | Full Bedrock model ID |
| `region` | Yes | AWS region |

LiteLLM calls Bedrock via IRSA. Set the IAM role ARN in `aws.litellmRoleArn` — the CLI annotates the LiteLLM service account automatically so pods assume this role instead of the EC2 node instance role. See [`aws`](#aws) below.

#### `models.openai[]`

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Display name |
| `model` | Yes | OpenAI model ID (e.g., `gpt-4o`) |
| `apiKey` | Yes | OpenAI API key (use `${ENV_VAR}`) |

#### `models.ollama[]`

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Display name |
| `model` | Yes | Ollama model ID (e.g., `llama3.2:3b`) |
| `apiBase` | Yes | URL of the Ollama server |

#### `models.azure[]`

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Display name |
| `model` | Yes | Deployment name |
| `apiBase` | Yes | Azure OpenAI endpoint URL |
| `apiVersion` | No | API version (default: `2024-02-01`) |
| `apiKey` | Yes | Azure API key |

### `postgresql` / `redis`

Leave all fields blank to use the in-cluster instances deployed by the Helm chart.

To use external datastores:

```yaml
postgresql:
  url: "postgresql://inferencehub:${POSTGRES_PASSWORD}@db.example.com:5432/inferencehub"
  username: inferencehub
  password: "${POSTGRES_PASSWORD}"
```

When `url` is set, the in-cluster StatefulSet is still deployed but not used by LiteLLM.

### `observability`

| Field | Default | Description |
|-------|---------|-------------|
| `enabled` | `false` | Enable Langfuse integration |
| `langfuse.host` | `https://cloud.langfuse.com` | Langfuse server URL |
| `langfuse.publicKey` | — | Langfuse public key |
| `langfuse.secretKey` | — | Langfuse secret key |

When `enabled: true`, all LiteLLM requests are traced in Langfuse.

### `aws`

Settings used when `cloudProvider: aws`. These are applied as Helm overrides — users never need to edit `values-aws.yaml`.

| Field | Required | Description |
|-------|----------|-------------|
| `litellmRoleArn` | Yes (if using Bedrock) | IAM Role ARN for the **LiteLLM service account**. Annotated as `eks.amazonaws.com/role-arn` so LiteLLM pods assume this role via IRSA instead of the EC2 node instance role. Format: `arn:aws:iam::<account-id>:role/<name>` |

The CLI annotates `litellm-sa` with this ARN automatically during `install` and `upgrade`. Without it, LiteLLM falls back to the node instance role — which typically lacks `bedrock:InvokeModel` permissions, causing all Bedrock calls to fail.

**How it works:**
1. The EKS pod identity webhook detects the `eks.amazonaws.com/role-arn` annotation on `litellm-sa`
2. On pod start it injects `AWS_ROLE_ARN` and `AWS_WEB_IDENTITY_TOKEN_FILE` env vars
3. LiteLLM uses the projected token to assume the role via STS and call Bedrock

If Bedrock models are configured but `litellmRoleArn` is not set, `inferencehub install` prints a warning.

> **Extensibility**: future providers (GCP, Azure) follow the same pattern — add a `gcp:` or `azure:` block with provider-specific fields.

## Environment variables

The CLI reads env vars in two ways:

1. **Auto-loaded .env files** (overrides in order):
   - `~/.inferencehub/.env`
   - `./.env`
   - `./.env.local`

2. **Interpolation** — any `${VAR_NAME}` in the config YAML is replaced with the value of `VAR_NAME` from the environment before parsing.

### Required

| Variable | Description |
|----------|-------------|
| `LITELLM_MASTER_KEY` | API key for the LiteLLM gateway. Must start with `sk-`. |

### Optional

| Variable | Description |
|----------|-------------|
| `POSTGRES_PASSWORD` | External PostgreSQL password |
| `REDIS_PASSWORD` | External Redis password |
| `LANGFUSE_PUBLIC_KEY` | Langfuse public key (when observability enabled) |
| `LANGFUSE_SECRET_KEY` | Langfuse secret key (when observability enabled) |
| `OPENAI_API_KEY` | OpenAI key (when using openai models) |
| `AZURE_OPENAI_API_KEY` | Azure key (when using azure models) |

## Example: AWS Bedrock deployment

```yaml
clusterName: prod-eks
domain: ai.company.com
environment: production
namespace: inferencehub
cloudProvider: aws        # auto-selects values-aws.yaml (gp3 storage)

gateway:
  name: inferencehub-gateway
  namespace: envoy-gateway-system

models:
  bedrock:
    - name: claude-sonnet
      model: anthropic.claude-3-5-sonnet-20241022-v2:0
      region: us-east-1
    - name: claude-haiku
      model: anthropic.claude-3-5-haiku-20241022-v1:0
      region: us-east-1

aws:
  litellmRoleArn: "arn:aws:iam::123456789012:role/litellm-bedrock-role"

observability:
  enabled: true
  langfuse:
    host: https://cloud.langfuse.com
    publicKey: "${LANGFUSE_PUBLIC_KEY}"
    secretKey: "${LANGFUSE_SECRET_KEY}"
```

Install with:

```bash
export LITELLM_MASTER_KEY="sk-..."
export LANGFUSE_PUBLIC_KEY="pk-lf-..."
export LANGFUSE_SECRET_KEY="sk-lf-..."

inferencehub install --config inferencehub.yaml
# values-aws.yaml is loaded automatically; IRSA ARN is injected from aws.irsaRoleArn
```

## Example: Local development (kind)

```yaml
clusterName: kind-local
domain: localhost
environment: staging
namespace: inferencehub
cloudProvider: local      # auto-selects values-local.yaml (reduced resources)

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

inferencehub install --config inferencehub.yaml
# values-local.yaml is loaded automatically
```

## Validating your config

```bash
inferencehub config validate --config inferencehub.yaml
```

This checks:
- Required fields are present
- `LITELLM_MASTER_KEY` is set and starts with `sk-`
- Model groups have required provider-specific fields
- External datastores have required connection fields

To see the fully-resolved config with env vars substituted:

```bash
inferencehub config show --config inferencehub.yaml
```
