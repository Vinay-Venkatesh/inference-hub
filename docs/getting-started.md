# Quick Start

InferenceHub is an infrastructure layer provisioner that helps you to create and manage your Internal AI platform via cli.

This guide walks you through on how to use inferencehub cli as your platform provisioner.

## Requirements

- Kubernetes cluster (see [cloud provider support](index.md#cloud-provider-support))
- `kubectl` and `helm` installed locally
- `go` installed (to build the CLI)
- `LITELLM_MASTER_KEY` environment variable set (must start with `sk-`)

---

## Step 1 — Install cluster prerequisites

Skip this step if your cluster already has **Gateway API CRDs**, **cert-manager**, **Envoy Gateway**, and a **Gateway resource**.

Otherwise, run once per cluster:

```bash
python3 scripts/setup-prerequisites.py \
  --cluster-name my-cluster \
  --domain inferencehub.ai \
  --environment staging \
  --tls-email admin@inferencehub.ai
```

This installs: Gateway API CRDs, cert-manager, Envoy Gateway, GatewayClass, Gateway, and optionally the AWS Load Balancer Controller.

See [Prerequisites](prerequisites.md) for full options and flags.

---

## Step 2 — Install the CLI locally

InferenceHub is written in Go. You can build and install it to your `$GOPATH/bin` (make sure your Go path is in your `$PATH`).

```bash
cd inferencehub-cli
make install
cd ..
```

Verify installation:
```bash
inferencehub --version
```

---

## Step 3 — Create a config file

Run all `inferencehub` commands from the **project root** (the directory that contains `helm/` and `scripts/`).

```bash
inferencehub config init
```

Edit the generated `inferencehub.yaml`:

```yaml
clusterName: my-cluster
domain: inferencehub.ai
environment: staging
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

observability:
  enabled: false
  langfuse:
    host: https://cloud.langfuse.com
    publicKey: "${LANGFUSE_PUBLIC_KEY}"
    secretKey: "${LANGFUSE_SECRET_KEY}"

# Optional: enable web search in OpenWebUI (deploys SearXNG in-cluster by default)
webSearch:
  enabled: true
```

See [Configuration Reference](configuration.md) for the full schema.

---

## Step 4 — Set environment variables

```bash
export LITELLM_MASTER_KEY="sk-your-secret-key"

# Optional — only needed if observability.enabled: true
export LANGFUSE_PUBLIC_KEY="pk-lf-..."
export LANGFUSE_SECRET_KEY="sk-lf-..."
```

The CLI auto-loads `.env`, `.env.local`, and `~/.inferencehub/.env`.

---

## Step 5 — Point DNS before installing

!!! warning "Do this before running install"
    cert-manager immediately attempts an HTTP-01 ACME challenge when you install. If your domain
    doesn't resolve to the load balancer yet, the challenge fails and cert-manager enters a backoff
    loop that requires manual intervention.

After the prerequisites script completes, get the NLB hostname:

```bash
kubectl get gateway inferencehub-gateway -n envoy-gateway-system \
  -o jsonpath='{.status.addresses[0].value}'
```

Create a **CNAME** record in your DNS provider:

```
your-domain.example.com  →  CNAME  →  <nlb-hostname>.elb.amazonaws.com
```

Verify propagation before continuing:

```bash
dig your-domain.example.com @8.8.8.8 +short
# Must return an IP or CNAME — if empty, wait and retry
```

---

## Step 6 — Install

```bash
inferencehub install --config inferencehub.yaml
```

---

## Step 7 — Verify

```bash
inferencehub verify
inferencehub status
```

---
