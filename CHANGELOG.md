# Changelog

## v0.2.1 — Test and Publish

**March 16 2026**

### What's new

**Test coverage for the CLI**
`GenerateOverrides` and config validation now have comprehensive unit test suites (22 and 15 test cases respectively). CI catches regressions automatically on every PR.

**Published Helm chart**
The InferenceHub Helm chart is now published to `oci://ghcr.io/vinay-venkatesh/inferencehub`. Install without cloning the repository:

```bash
helm install inferencehub oci://ghcr.io/vinay-venkatesh/inferencehub --version 0.2.1 -n inferencehub --create-namespace -f values.yaml
```

**Helm-only install guide**
ArgoCD, Flux, and GitOps users can now install InferenceHub without the CLI. See `docs/helm-only-install.md` for the complete guide including ArgoCD Application and Flux HelmRelease examples.

**Automated release workflow**
Git tag pushes (`v*`) now automatically run tests, package and publish the Helm chart to ghcr.io, build CLI binaries for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, and create a GitHub Release with binaries attached.

**CI enhancements**
PR CI now also runs `helm template --strict` (catches rendering errors that `helm lint` misses) and `golangci-lint`.

---

## v0.2.0 — Foundation

**March 07 2026**

### Breaking changes

- **Database split**: the single `inferencehub` PostgreSQL database is now two separate databases — `openwebui` and `litellm`. Existing installations must run the migration script (`docs/upgrading/migrate-database.sh`) before upgrading.
- **External PostgreSQL config**: `postgresql.external.connectionString` is replaced by `postgresql.external.openwebuiConnectionString` and `postgresql.external.litellmConnectionString`.

### What's new

**Web search with flexible engine support**
OpenWebUI can now perform web searches directly from the chat interface. InferenceHub deploys [SearXNG](https://github.com/searxng/searxng) in-cluster by default — no external account or API key required. Users who already have a search engine can bring their own instead.

Enable with a single config flag:

```yaml
webSearch:
  enabled: true   # deploys SearXNG in-cluster automatically
```

To use an external engine:

```yaml
webSearch:
  enabled: true
  engine: brave          # brave | bing | tavily | google_pse | duckduckgo | searxng
  external:
    enabled: true
    apiKey: "${BRAVE_API_KEY}"
    # queryUrl: "..."    # for external SearXNG instances
    # engineId: "..."    # for google_pse only
```

**Upstream subcharts replace custom templates**
OpenWebUI and LiteLLM are now deployed via their official Helm subcharts:
- `open-webui` chart `12.5.0` (from `https://helm.openwebui.com/`)
- `litellm-helm` chart `1.81.12-stable` (from `oci://ghcr.io/berriai`)

This gives users the full feature set of each upstream chart — SSO, web search, pipelines, alerting, etc. — by adding values to `inferencehub.yaml` without waiting for InferenceHub to expose each option.

**Full passthrough configuration**
Add an `openwebui:` or `litellm:` block to `inferencehub.yaml` to pass any value directly to the upstream chart:

```yaml
openwebui:
  defaultUserRole: pending   # require admin approval for new users

litellm:
  proxy_config:
    litellm_settings:
      request_timeout: 600
```

InferenceHub's required injections (database URLs, Redis wiring, master key, model list) are always applied on top.

**LiteLLM connected to PostgreSQL by default**
LiteLLM now uses the `litellm` database out of the box, which is required for virtual keys, teams, and spend tracking for the future releases.

**LiteLLM image switched to `litellm-database` variant**
Uses `ghcr.io/berriai/litellm-database` which bundles Prisma and database migration support.

### Upgrade

See `docs/upgrading/v0.1-to-v0.2.md` in the repository for the full step-by-step guide.

---

## v0.1.0 — Initial Release

**March 01 2026**

Open-sourcing InferenceHub — a Kubernetes-native LLM control plane that deploys a complete, self-hosted AI inference stack with a single CLI command.

### What is InferenceHub?

InferenceHub eliminates the operational overhead of running LLM infrastructure on Kubernetes. Instead of manually wiring together a chat interface, model gateway, database, cache, TLS, and observability — InferenceHub provisioner ship them as a single opinionated platform, configured through one YAML file and installed with one command.

```
inferencehub install --config inferencehub.yaml
```

---

### What's included

| Component | Version | Role |
|-----------|---------|------|
| OpenWebUI | `v0.8.5` | ChatGPT-style web interface |
| LiteLLM | `v1.81.12` | OpenAI-compatible API gateway |
| PostgreSQL | `18` | Persistent storage |
| Redis | `8` | Response caching |
| Envoy Gateway | `v1.7.0` | Kubernetes Gateway API implementation |
| cert-manager | `v1.19.4` | Automatic TLS via Let's Encrypt |
| Gateway API CRDs | `v1.5.0` | Kubernetes networking extension |

---

### Features

**Single-command installation**
One CLI, one config file. `inferencehub install` handles namespace creation, Helm values generation, pod readiness, and post-install verification.

**Multi-provider model routing**
Route to AWS Bedrock, OpenAI, Azure OpenAI, and self-hosted Ollama models — all through a unified OpenAI-compatible API. Configure all providers in one `models:` block.

**AWS EKS native**
First-class AWS support: automatic gp3 storage class detection, IRSA annotation for the LiteLLM service account (Bedrock access without hardcoded credentials), and internet-facing NLB provisioning via Envoy Gateway.

**Automatic TLS**
cert-manager with Let's Encrypt HTTP-01 via Gateway API. Staging and production issuers selected automatically from `environment: staging|production` in config.

**Declarative configuration**
A single `inferencehub.yaml` drives the entire platform. Secrets stay out of the file via `${ENV_VAR}` interpolation. External datastores (RDS, ElastiCache) and observability (Langfuse) are opt-in with a few extra lines.

**Idempotent prerequisites script**
`setup-prerequisites.py` installs Gateway API CRDs, cert-manager, Envoy Gateway, and the AWS Load Balancer Controller. Every step is skipped if already present — safe to re-run.

**CLI-driven lifecycle**
`install`, `upgrade`, `status`, `verify`, `uninstall` — full lifecycle management from the terminal.

---

### Supported model providers

| Provider | Notes |
|----------|-------|
| AWS Bedrock | Requires IRSA role ARN via `aws.litellmRoleArn` |
| OpenAI | Any OpenAI-compatible API |
| Azure OpenAI | Per-deployment configuration |
| Ollama | Self-hosted, any accessible endpoint |

---

### Platform support

| Platform | Status |
|----------|--------|
| AWS EKS | ✅ Tested and supported |
| Local / kind | ⚠️ Best effort |
| GKE / AKS | 🔜 Roadmap |

---

### Known limitations

- **AWS EKS only** for production use in this release. GKE and AKS support is planned.
- **In-cluster PostgreSQL** is not recommended for production — use an external managed database (RDS). A warning is shown at install time.
- **DNS must be configured before running `inferencehub install`** — cert-manager fires the HTTP-01 ACME challenge immediately on install. If the domain doesn't resolve at that point, the certificate enters a failed state that requires manual recovery.

---

### Getting started

```bash
# 1. Install prerequisites (once per cluster)
python3 scripts/setup-prerequisites.py \
  --cluster-name my-cluster \
  --domain inferencehub.ai \
  --tls-email admin@inferencehub.ai \
  --aws-lb-role-arn arn:aws:iam::123456789012:role/AWSLoadBalancerControllerRole

# 2. Point DNS to the NLB before proceeding
kubectl get gateway inferencehub-gateway -n envoy-gateway-system \
  -o jsonpath='{.status.addresses[0].value}'

# 3. Install
export LITELLM_MASTER_KEY="sk-..."
inferencehub install --config inferencehub.yaml
```

Full documentation: [Getting-Started](getting-started.md)
