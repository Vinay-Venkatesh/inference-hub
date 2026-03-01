# InferenceHub Helm Chart

The Helm chart for [InferenceHub](https://github.com/Vinay-Venkatesh/inference-hub) — a Kubernetes-native LLM control plane that deploys a complete, self-hosted AI inference stack.

> **This chart is designed to be installed via the `inferencehub` CLI**, which handles values generation, IRSA annotation, and post-install verification automatically. Direct `helm install` usage is supported but requires manual values configuration.

## Components

| Component | Image | Version |
|-----------|-------|---------|
| OpenWebUI | `ghcr.io/open-webui/open-webui` | `v0.8.5` |
| LiteLLM | `ghcr.io/berriai/litellm` | `main-v1.81.12-stable.2` |
| PostgreSQL | `postgres` | `18-alpine` |
| Redis | `redis` | `8-alpine` |

Cluster-wide dependencies (Gateway API CRDs, cert-manager, Envoy Gateway) are **not** bundled in this chart. Install them once per cluster using `scripts/setup-prerequisites.py`.

## Recommended: Install via CLI

```bash
inferencehub install --config inferencehub.yaml
```

The CLI reads your `inferencehub.yaml`, generates the correct Helm values, applies cloud provider presets (e.g. `values-aws.yaml` when `cloudProvider: aws`), and annotates the LiteLLM service account for IRSA.

See the [project README](../../README.md) for the full quick start guide.

## Direct Helm Install

If you prefer to install without the CLI:

```bash
helm install inferencehub ./helm/inferencehub \
  --create-namespace \
  --namespace inferencehub \
  --set litellm.masterKey="sk-your-secret-key" \
  -f helm/inferencehub/values-aws.yaml   # or values-local.yaml
```

For AWS EKS, also annotate the LiteLLM service account for Bedrock IRSA:

```bash
helm install inferencehub ./helm/inferencehub \
  --create-namespace \
  --namespace inferencehub \
  --set litellm.masterKey="sk-your-secret-key" \
  --set litellm.serviceAccount.annotations."eks\.amazonaws\.com/role-arn"="arn:aws:iam::123456789012:role/litellm-bedrock-role" \
  -f helm/inferencehub/values-aws.yaml
```

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
| `litellm.models` | Model list passed to LiteLLM `config.yaml` | `[]` |
| `litellm.serviceAccount.annotations` | IRSA annotation for Bedrock access | `{}` |
| `postgresql.external.enabled` | Use an external PostgreSQL (e.g. RDS) | `false` |
| `postgresql.external.connectionString` | External DB connection string | `""` |
| `redis.external.enabled` | Use an external Redis (e.g. ElastiCache) | `false` |
| `networking.hostname` | Public hostname for HTTPRoute | `""` |
| `networking.gatewayRef.name` | Envoy Gateway name | `inferencehub-gateway` |
| `networking.gatewayRef.namespace` | Envoy Gateway namespace | `envoy-gateway-system` |
| `networking.tls.certIssuer` | cert-manager ClusterIssuer name | `letsencrypt-staging` |
| `observability.langfuse.enabled` | Enable Langfuse tracing | `false` |

See `values.yaml` for the full parameter reference.

## External Datastores

### External PostgreSQL (RDS)

```yaml
postgresql:
  enabled: true
  external:
    enabled: true
    connectionString: "postgresql://user:pass@mydb.cluster.us-east-1.rds.amazonaws.com:5432/inferencehub"
```

> In-cluster PostgreSQL is not recommended for production. Use RDS or another managed database.

### External Redis (ElastiCache)

```yaml
redis:
  enabled: true
  external:
    enabled: true
    host: "my-cache.abc123.cache.amazonaws.com"
    port: 6379
    password: ""
```

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
