# Changelog

## v0.1.0 — Initial Release

**March 2026**

Open-sourcing InferenceHub — a Kubernetes-native LLM control plane that deploys a complete, self-hosted AI inference stack with a single CLI command.

### What is InferenceHub?

InferenceHub eliminates the operational overhead of running LLM infrastructure on Kubernetes. Instead of manually wiring together a chat interface, model gateway, database, cache, TLS, and observability — InferenceHub ships them as a single opinionated platform, configured through one YAML file and installed with one command.

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

### Roadmap

- GKE and AKS support
- Helm OCI registry (`helm install` without cloning the repo)
- Multi-tenant isolation (namespace-per-team, separate API keys)
- RAG primitives (pgvector / Weaviate integration)

---

### Getting started

```bash
# 1. Install prerequisites (once per cluster)
python3 scripts/setup-prerequisites.py \
  --cluster-name my-cluster \
  --domain ai.example.com \
  --tls-email admin@example.com \
  --aws-lb-role-arn arn:aws:iam::123456789012:role/AWSLoadBalancerControllerRole

# 2. Point DNS to the NLB before proceeding
kubectl get gateway inferencehub-gateway -n envoy-gateway-system \
  -o jsonpath='{.status.addresses[0].value}'

# 3. Install
export LITELLM_MASTER_KEY="sk-..."
inferencehub install --config inferencehub.yaml
```

Full documentation: [docs/configuration.md](docs/configuration.md) · [docs/prerequisites.md](docs/prerequisites.md)
