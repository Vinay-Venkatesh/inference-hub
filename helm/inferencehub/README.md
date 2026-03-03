# InferenceHub Helm Chart

The Helm chart for [InferenceHub](https://github.com/Vinay-Venkatesh/inference-hub) — a Kubernetes-native LLM control plane that deploys a complete, self-hosted AI inference stack.

> **This chart is designed to be installed via the `inferencehub` CLI**, which handles values generation, IRSA annotation, and post-install verification automatically. Direct `helm install` usage is supported but requires manual values configuration.

## Components

| Component | Image | Version | Source |
|-----------|-------|---------|--------|
| OpenWebUI | `ghcr.io/open-webui/open-webui` | `v0.8.5` | [open-webui chart](https://helm.openwebui.com/) `12.5.0` |
| LiteLLM | `ghcr.io/berriai/litellm-database` | `v1.81.12-stable` | [litellm-helm chart](https://github.com/BerriAI/litellm/tree/main/deploy/charts/litellm-helm) `1.81.12-stable` |
| PostgreSQL | `postgres` | `18-alpine` | InferenceHub managed |
| Redis | `redis` | `8-alpine` | InferenceHub managed |

Cluster-wide dependencies (Gateway API CRDs, cert-manager, Envoy Gateway) are **not** bundled in this chart. Install them once per cluster using `scripts/setup-prerequisites.py`.

## Recommended: Install via CLI

```bash
inferencehub install --config inferencehub.yaml
```

The CLI reads your `inferencehub.yaml`, generates the correct Helm values, applies cloud provider presets (e.g. `values-aws.yaml` when `cloudProvider: aws`), and annotates the LiteLLM service account for IRSA.

See the [project README](../../README.md) for the full quick start guide.

## Direct Helm Install

If you prefer to install without the CLI, first ensure subchart dependencies are present:

```bash
cd helm/inferencehub
helm dep build   # downloads open-webui and litellm-helm into charts/
```

Then install:

```bash
helm install inferencehub ./helm/inferencehub \
  --create-namespace \
  --namespace inferencehub \
  --set litellm.masterKey="sk-your-secret-key" \
  --set postgresql.auth.password="your-db-password" \
  --set networking.gatewayAPI.hostname="inferencehub.example.com" \
  -f helm/inferencehub/values-aws.yaml   # or values-local.yaml
```

For AWS EKS with Bedrock IRSA:

```bash
helm install inferencehub ./helm/inferencehub \
  --create-namespace \
  --namespace inferencehub \
  --set litellm.masterKey="sk-your-secret-key" \
  --set postgresql.auth.password="your-db-password" \
  --set networking.gatewayAPI.hostname="inferencehub.example.com" \
  --set "litellm.serviceAccount.annotations.eks\.amazonaws\.com/role-arn=arn:aws:iam::123456789012:role/litellm-bedrock-role" \
  -f helm/inferencehub/values-aws.yaml
```

> **Note:** Direct `helm install` skips the model list generation. Use the CLI (`inferencehub install`) to have models from `inferencehub.yaml` automatically translated into LiteLLM's `proxy_config.model_list`.

## Values Files

| File | Purpose |
|------|---------|
| `values.yaml` | Base defaults — all components, all settings |
| `values-aws.yaml` | AWS preset: gp3 storage class (auto-selected by CLI when `cloudProvider: aws`) |
| `values-local.yaml` | Local/kind overrides: no storage class, NodePort services |

## Key Configuration Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `litellm.masterKey` | LiteLLM API key (must start with `sk-`) | `""` |
| `litellm.serviceAccount.annotations` | IRSA annotation for Bedrock access | `{}` |
| `postgresql.external.enabled` | Use an external PostgreSQL (e.g. RDS) | `false` |
| `postgresql.external.openwebuiConnectionString` | Connection string for the OpenWebUI database | `""` |
| `postgresql.external.litellmConnectionString` | Connection string for the LiteLLM database | `""` |
| `redis.openwebui.external.enabled` | Use external Redis for OpenWebUI (e.g. ElastiCache) | `false` |
| `redis.litellm.external.enabled` | Use external Redis for LiteLLM (e.g. ElastiCache) | `false` |
| `networking.gatewayAPI.hostname` | Public hostname for HTTPRoute | `""` |
| `networking.gatewayAPI.gatewayRef.name` | Envoy Gateway name | `inferencehub-gateway` |
| `networking.gatewayAPI.gatewayRef.namespace` | Envoy Gateway namespace | `envoy-gateway-system` |
| `networking.gatewayAPI.tls.issuerRef` | cert-manager ClusterIssuer name | `letsencrypt-staging` |
| `observability.langfuse.enabled` | Enable Langfuse tracing | `false` |

See `values.yaml` for the full parameter reference.

## Passthrough Values

`openwebui:` and `litellm:` keys pass directly to the upstream subcharts. Any value supported by the upstream chart can be set here — InferenceHub merges its required injections on top.

```yaml
# openwebui: any value from https://github.com/open-webui/helm-charts/blob/main/charts/open-webui/values.yaml
openwebui:
  defaultUserRole: pending     # example: require admin approval for new signups

# litellm: any value from https://github.com/BerriAI/litellm/blob/main/deploy/charts/litellm-helm/values.yaml
litellm:
  proxy_config:
    litellm_settings:
      request_timeout: 600
```

The following keys are **always overridden by InferenceHub** — do not set them:

| Key | Reason |
|-----|--------|
| `openwebui.openaiBaseApiUrl` | Wired to LiteLLM service |
| `openwebui.extraEnvVars[DATABASE_URL]` | Injected from PostgreSQL secret |
| `openwebui.extraEnvVars[OPENAI_API_KEY]` | Injected from LiteLLM master key secret |
| `openwebui.ollama.enabled` | Always `false` — use `models.ollama` with an external URL instead |
| `openwebui.websocket.redis.enabled` | Always `false` — InferenceHub provides a dedicated Redis for OpenWebUI (`redis.openwebui`) |
| `litellm.masterkeySecretName` | Points to InferenceHub's managed secret |
| `litellm.db.deployStandalone` | InferenceHub provides shared PostgreSQL |
| `litellm.proxy_config.model_list` | Generated from `inferencehub.yaml` `models:` section |
| `litellm.environmentSecrets` | InferenceHub appends its wiring secret |

## External Datastores

### External PostgreSQL (RDS)

v0.2.0 uses separate databases for OpenWebUI and LiteLLM. Provide two connection strings:

```yaml
postgresql:
  enabled: false
  external:
    enabled: true
    openwebuiConnectionString: "postgresql://user:pass@mydb.cluster.us-east-1.rds.amazonaws.com:5432/openwebui"
    litellmConnectionString:   "postgresql://user:pass@mydb.cluster.us-east-1.rds.amazonaws.com:5432/litellm"
```

> In-cluster PostgreSQL is not recommended for production. Use RDS or another managed database.

### External Redis (ElastiCache)

InferenceHub deploys two separate Redis instances — one for OpenWebUI session state, one for LiteLLM API caching — to avoid eviction policy conflicts. Each can independently point to an external Redis.

```yaml
redis:
  openwebui:
    enabled: false
    external:
      enabled: true
      host: "openwebui-cache.abc123.cache.amazonaws.com"
      port: 6379
      password: ""      # or set OPENWEBUI_REDIS_PASSWORD env var
  litellm:
    enabled: false
    external:
      enabled: true
      host: "litellm-cache.def456.cache.amazonaws.com"
      port: 6379
      password: ""      # or set LITELLM_REDIS_PASSWORD env var
```

You can also mix in-cluster and external — leave one block at defaults and configure only the other.

See [docs/infrastructure.md](../../docs/infrastructure.md) for the rationale behind separate Redis instances.

## Upgrade

```bash
# Via CLI (recommended)
inferencehub upgrade --config inferencehub.yaml

# Via Helm directly
helm upgrade inferencehub ./helm/inferencehub \
  --namespace inferencehub \
  -f helm/inferencehub/values-aws.yaml
```

## Uninstall

```bash
# Via CLI
inferencehub uninstall --confirm

# Via Helm directly
helm uninstall inferencehub --namespace inferencehub
```

This removes InferenceHub workloads and services. Cluster-wide components (cert-manager, Envoy Gateway) are not touched — remove them separately with `scripts/uninstall-prerequisites.py` if needed.

## Links

- [Project README](../../README.md)
- [Configuration reference](../../docs/configuration.md)
- [Prerequisites setup](../../docs/prerequisites.md)
- [GitHub](https://github.com/Vinay-Venkatesh/inference-hub)
