# InferenceHub

InferenceHub is a Kubernetes-native LLM control plane that deploys a ChatGPT-style interface with unified multi-provider model routing. <br/>

Self-hosted. Open-source. Designed for Kubernetes environments. <br/>

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## What it includes

| Component | Version | Role |
|-----------|---------|------|
| [OpenWebUI](https://github.com/open-webui/open-webui) | `v0.8.5` | ChatGPT-style web interface for interacting with LLMs |
| [LiteLLM](https://github.com/BerriAI/litellm) | `v1.81.12` | OpenAI-compatible API gateway routing to 2000+ model providers |
| [PostgreSQL](https://hub.docker.com/_/postgres) | `18` | Persistent storage for users, conversations, and configuration |
| [Redis](https://hub.docker.com/_/redis) | `8` | Optional response caching layer to reduce API costs and latency |
| [Envoy Gateway](https://github.com/envoyproxy/gateway) | `v1.7.0` | Kubernetes Gateway API implementation |
| [cert-manager](https://github.com/cert-manager/cert-manager) | `v1.19.4` | Automatic TLS via Let's Encrypt |
| [Langfuse](https://langfuse.com) | SaaS | for LLM observability and cost tracking |

## Why InferenceHub?

Running LLM tools inside Kubernetes often requires stitching together a UI layer, API gateway, storage, caching, and observability — each configured separately. <br/>

InferenceHub standardizes this stack into a single, opinionated deployment, reducing operational overhead and configuration drift. <br/>

This project is early-stage and aims to evolve into a declarative internal AI platform for organizations running LLM workloads on Kubernetes. <br/>


## Vision

>*InferenceHub aims to provide declarative model management, team isolation, and first-class RAG primitives as part of the platform.*

## How it works

![overview](./docs/images/overview.png)

Routing is handled by the Kubernetes Gateway API (HTTPRoute). TLS is managed by cert-manager with Let's Encrypt. The Helm chart deploys only the application layer — cluster-wide components (Gateway, cert-manager) are installed separately via the prerequisites script.

## Cloud provider support

> **v0.1.0 is tested and supported on AWS EKS.** The Helm chart is cloud-agnostic, but the prerequisites script, NLB configuration, IRSA integration, and storage class defaults are built and validated for AWS. Other providers are on the roadmap.

| Provider | Status | Notes |
|----------|--------|-------|
| **AWS EKS** | ✅ Supported | NLB via Envoy Gateway, IRSA for Bedrock, gp3 storage |
| GKE | 🔜 Planned | Cloud Load Balancer, Workload Identity |
| AKS | 🔜 Planned | Azure Load Balancer, Workload Identity |
| Local / kind | ⚠️ Best effort | No cloud-specific features; works for development |

## Roadmap

- [ ] **GKE support** — GKE Autopilot compatibility, Workload Identity for Vertex AI, Cloud SQL / Memorystore as external datastores
- [ ] **AKS support** — Azure Load Balancer, Managed Identity for Azure OpenAI, Azure Database for PostgreSQL
- [ ] **Helm OCI registry** — publish chart to `ghcr.io` so users can `helm install` without cloning the repo
- [ ] **Multi-tenant isolation** (namespace-per-team, separate API keys)
- [ ] **RAG primitives** (pgvector and Weaviate integration)

>*Watch out this section for more updates*

## Requirements

- Kubernetes cluster (EKS recommended; see [cloud provider support](#cloud-provider-support) above)
- `kubectl` and `helm` installed locally
- `LITELLM_MASTER_KEY` environment variable set (must start with `sk-`)

## Quick start

### 1. Install cluster prerequisites _(skip if already present)_

If your cluster already has **Gateway API CRDs**, **cert-manager**, **Envoy Gateway**, and a **Gateway resource**, skip this step and go straight to step 2.

Otherwise, run once per cluster:

```
python3 scripts/setup-prerequisites.py \
  --cluster-name my-cluster \
  --domain inferencehub.ai \
  --environment staging \
  --tls-email admin@inferencehub.ai
```

This installs: Gateway API CRDs, cert-manager, Envoy Gateway, GatewayClass, Gateway, and optionally the AWS Load Balancer Controller.

See [docs/prerequisites.md](docs/prerequisites.md) for full options.

### 2. Create a config file

Run all `inferencehub` commands from the **project root** (the directory that contains `helm/` and `scripts/`). The CLI creates `inferencehub.yaml` and reads `.env` from your current directory.

```bash
# Make sure you're at the project root
cd /path/to/inference-platform

inferencehub config init
```

Edit the generated `inferencehub.yaml`:

```yaml
clusterName: my-cluster
domain: ai.example.com
environment: staging
namespace: inferencehub
cloudProvider: aws        # auto-selects helm/inferencehub/values-aws.yaml

gateway:
  name: inferencehub-gateway
  namespace: envoy-gateway-system

models:
  bedrock:
    - name: claude-sonnet
      model: anthropic.claude-3-5-sonnet-20241022-v2:0
      region: us-east-1

# IAM role for the LiteLLM service account — required for Bedrock access
aws:
  litellmRoleArn: "arn:aws:iam::123456789012:role/litellm-bedrock-role"

observability:
  enabled: false
  langfuse:
    host: https://cloud.langfuse.com
    publicKey: "${LANGFUSE_PUBLIC_KEY}"
    secretKey: "${LANGFUSE_SECRET_KEY}"
```

See [docs/configuration.md](docs/configuration.md) for the full schema.

### 3. Set environment variables

```bash
export LITELLM_MASTER_KEY="sk-your-secret-key"
# optional
export LANGFUSE_PUBLIC_KEY="pk-lf-..."
export LANGFUSE_SECRET_KEY="sk-lf-..."
```

Use a `.env` file — the CLI auto-loads `.env`, `.env.local`, and `~/.inferencehub/.env`.

### 4. Point DNS before installing

> [!CAUTION]
> **Set up your DNS record before running `inferencehub install`.** The installer deploys cert-manager, which immediately attempts an HTTP-01 ACME challenge to issue a TLS certificate. If your domain does not resolve to the load balancer at that point, the challenge fails and cert-manager enters a backoff loop that requires manual intervention to clear.

After the prerequisites script completes, get the NLB hostname:

```bash
kubectl get gateway inferencehub-gateway -n envoy-gateway-system \
  -o jsonpath='{.status.addresses[0].value}'
```

Then in your DNS provider, create a **CNAME** record before proceeding:

```
inferencehub.platformaiq.com  →  CNAME  →  <nlb-hostname>.elb.amazonaws.com
```

Verify propagation before continuing:

```bash
dig inferencehub.platformaiq.com @8.8.8.8 +short
# Must return an IP or CNAME — if empty, wait and retry
```

### 5. Install

```bash
inferencehub install --config inferencehub.yaml
```

When `cloudProvider: aws` is set in your config, the CLI automatically uses `helm/inferencehub/values-aws.yaml` (gp3 storage) and annotates the LiteLLM service account with `aws.litellmRoleArn` for Bedrock IRSA — no manual values file editing required.

### 6. Verify

```bash
inferencehub verify
inferencehub status
```

## Troubleshooting: site not accessible after install

If pods are running but your domain isn't loading, the most common cause is a **TLS certificate that hasn't been issued yet**. Work through these checks in order.

### Check certificate status

```bash
kubectl get certificate -n inferencehub
```

The `READY` column must be `True`. If it shows `False`, continue below.

### Check the full issuance chain

```bash
kubectl get certificate,certificaterequest,order,challenge -n inferencehub
```

Every resource in this chain must reach a terminal ready/approved state:

| Resource | Healthy state |
|----------|--------------|
| `Certificate` | `READY = True` |
| `CertificateRequest` | `APPROVED = True`, `READY = True` |
| `Order` | `STATE = valid` |
| `Challenge` | `STATE = valid` (deleted once complete) |

### Diagnose a stuck resource

```bash
# Describe whichever resource is not healthy
kubectl describe certificate inferencehub-tls -n inferencehub
kubectl describe certificaterequest -n inferencehub
kubectl describe order -n inferencehub
kubectl describe challenge -n inferencehub
```

The `Events` section at the bottom will show the exact failure reason.

### Common failure: DNS not propagated

```
DNS problem: NXDOMAIN looking up A for <domain>
```

The ACME server could not resolve your domain. Check:

```bash
dig <your-domain> @8.8.8.8 +short   # must return an IP or CNAME
```

If empty, your DNS record hasn't propagated yet. Wait and then force a retry:

```bash
kubectl delete certificaterequest -n inferencehub --all
# cert-manager recreates it automatically within seconds
```

### Common failure: order stuck in invalid state

```bash
# Delete the failed order — cert-manager recreates it from the CertificateRequest
kubectl delete order -n inferencehub --all
```

If cert-manager does not recreate the CertificateRequest automatically, delete the Certificate and re-apply:

```bash
kubectl delete certificate inferencehub-tls -n inferencehub
inferencehub upgrade --config inferencehub.yaml
```

## CLI reference

| Command | Description |
|---------|-------------|
| `inferencehub config init` | Generate a starter config file |
| `inferencehub config validate --config <file>` | Validate config and env vars |
| `inferencehub config show --config <file>` | Show config after env var interpolation |
| `inferencehub install --config <file>` | Install the platform |
| `inferencehub upgrade --config <file>` | Upgrade an existing installation |
| `inferencehub status` | Show component health |
| `inferencehub verify` | Check prerequisites and platform |
| `inferencehub uninstall --confirm` | Remove the platform |

Run `inferencehub <command> --help` for all flags.

## Supported model providers

| Provider | Example model ID |
|----------|-----------------|
| AWS Bedrock | `anthropic.claude-3-5-sonnet-20241022-v2:0` |
| OpenAI | `gpt-4o` |
| Ollama (self-hosted) | `llama3.2:3b` |
| Azure OpenAI | `gpt-4` |

## Project layout

```
inference-platform/
├── helm/
│   └── inferencehub/          # The Helm chart
│       ├── Chart.yaml
│       ├── values.yaml         # Defaults
│       ├── values-aws.yaml     # AWS preset (gp3 storage) — auto-selected, do not edit
│       ├── values-local.yaml   # Local/kind overrides
│       └── templates/
│           ├── openwebui/
│           ├── litellm/
│           ├── postgresql/
│           ├── redis/
│           └── networking/     # HTTPRoute, Certificate, ReferenceGrant
├── scripts/
│   ├── setup-prerequisites.py    # Install cluster-wide components
│   └── uninstall-prerequisites.py
├── inferencehub-cli/             # CLI source code (Go)
├── examples/
│   ├── config-aws.yaml
│   ├── config-local.yaml
│   └── .env.example
└── docs/
    ├── prerequisites.md
    ├── configuration.md
    └── implementation-plan-v1.md
```

## Documentation

- [Prerequisites setup](docs/prerequisites.md) — Install cluster-wide components
- [Configuration reference](docs/configuration.md) — Full `config.yaml` schema
- [AWS deployment](docs/prerequisites.md#aws-load-balancer-controller) — EKS-specific setup


## Demo

[![asciicast](https://asciinema.org/a/3mcWdditQI1ZlNlU.svg)](https://asciinema.org/a/3mcWdditQI1ZlNlU)


## LangFuse integration

![overview](./docs/images/langfuse.png)

## Contributing

1. Fork the repository
2. Create a feature branch
3. Run `helm lint helm/inferencehub/` to validate chart changes
4. Run `cd inferencehub-cli && go build ./...` to validate CLI changes
5. Open a pull request

Refer to [Guide](./CONTRIBUTING.md) for more details

## License

MIT — see [LICENSE](LICENSE).
