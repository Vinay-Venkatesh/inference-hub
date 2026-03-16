# Helm-only Install

This guide covers installing InferenceHub directly with Helm, without the `inferencehub` CLI. This is the recommended approach for GitOps workflows (ArgoCD, Flux) or any environment where you want declarative, version-controlled Helm values rather than a CLI-driven install.

## When to use Helm-only vs CLI

| | CLI install | Helm-only |
|---|---|---|
| **Recommended for** | First-time setup, local dev, quick deploys | GitOps, ArgoCD, Flux, CI/CD pipelines |
| **Values generation** | Automatic (CLI generates all required overrides) | Manual (you provide values.yaml) |
| **Secret wiring** | Automatic | Manual (follow this guide) |
| **Upgrade** | `inferencehub upgrade` | `helm upgrade` |

## Prerequisites

Before installing, run the prerequisites script to install the required cluster components (Gateway API CRDs, cert-manager, Envoy Gateway, and the AWS Load Balancer Controller on EKS):

```bash
python3 scripts/setup-prerequisites.py \
  --cluster-name my-cluster \
  --domain inference.example.com \
  --tls-email admin@example.com \
  --aws-lb-role-arn arn:aws:iam::123456789012:role/AWSLoadBalancerControllerRole
```

Point your domain's DNS to the NLB address before proceeding:

```bash
kubectl get gateway inferencehub-gateway -n envoy-gateway-system \
  -o jsonpath='{.status.addresses[0].value}'
```

## Installation

Install InferenceHub from the OCI registry without cloning the repository:

```bash
helm install inferencehub oci://ghcr.io/vinay-venkatesh/inferencehub \
  --version 0.2.1 \
  --namespace inferencehub \
  --create-namespace \
  -f values.yaml
```

## Required values

The CLI auto-generates the following wiring values. When installing with Helm directly, you must provide them manually in your `values.yaml`.

### Networking

```yaml
networking:
  gatewayAPI:
    hostname: inference.example.com
    gatewayRef:
      name: inferencehub-gateway
      namespace: envoy-gateway-system
    tls:
      issuerRef: letsencrypt-prod   # or letsencrypt-staging
```

### LiteLLM wiring

```yaml
litellm:
  # Master key — must match the LITELLM_MASTER_KEY env var in the litellm-env secret
  masterKey: "sk-your-master-key-here"
  masterkeySecretName: inferencehub-litellm-secret
  masterkeySecretKey: master-key

  # Disable bundled database (use InferenceHub's PostgreSQL)
  db:
    deployStandalone: false
    useExisting: true
    endpoint: inferencehub-postgresql
    database: litellm
    secret:
      name: inferencehub-postgresql-secret
      usernameKey: postgres-user
      passwordKey: postgres-password

  # Disable bundled Redis (InferenceHub wires Redis via environmentSecrets)
  redis:
    enabled: false

  # Required wiring secret that provides DATABASE_URL and REDIS_* to LiteLLM
  environmentSecrets:
    - inferencehub-litellm-env

  # Model routing configuration
  proxy_config:
    model_list:
      - model_name: claude-3-sonnet
        litellm_params:
          model: bedrock/anthropic.claude-3-sonnet-20240229-v1:0
          aws_region_name: us-east-1
    general_settings:
      master_key: "os.environ/LITELLM_MASTER_KEY"
```

### OpenWebUI wiring

```yaml
openwebui:
  # Always disable — InferenceHub routes through LiteLLM, not Ollama subchart
  ollama:
    enabled: false

  # Always disable — InferenceHub provides a dedicated Redis for websockets
  websocket:
    redis:
      enabled: false
    url: redis://inferencehub-redis-openwebui:6379/0

  # Point OpenWebUI at the LiteLLM gateway
  openaiBaseApiUrl: http://inferencehub-litellm:4000/v1

  # Wire DATABASE_URL and OPENAI_API_KEY from secrets
  extraEnvVars:
    - name: DATABASE_URL
      valueFrom:
        secretKeyRef:
          name: inferencehub-postgresql-secret
          key: openwebui-database-url
    - name: OPENAI_API_KEY
      valueFrom:
        secretKeyRef:
          name: inferencehub-litellm-secret
          key: master-key
```

## Example values.yaml

Complete working example for AWS EKS with Bedrock models and IRSA:

```yaml
networking:
  gatewayAPI:
    hostname: inference.example.com
    gatewayRef:
      name: inferencehub-gateway
      namespace: envoy-gateway-system
    tls:
      issuerRef: letsencrypt-prod

litellm:
  masterKey: "sk-your-master-key-here"
  masterkeySecretName: inferencehub-litellm-secret
  masterkeySecretKey: master-key

  db:
    deployStandalone: false
    useExisting: true
    endpoint: inferencehub-postgresql
    database: litellm
    secret:
      name: inferencehub-postgresql-secret
      usernameKey: postgres-user
      passwordKey: postgres-password

  redis:
    enabled: false

  environmentSecrets:
    - inferencehub-litellm-env

  serviceAccount:
    annotations:
      eks.amazonaws.com/role-arn: arn:aws:iam::123456789012:role/LiteLLMBedrockRole

  proxy_config:
    model_list:
      - model_name: claude-3-sonnet
        litellm_params:
          model: bedrock/anthropic.claude-3-sonnet-20240229-v1:0
          aws_region_name: us-east-1
      - model_name: claude-3-haiku
        litellm_params:
          model: bedrock/anthropic.claude-3-haiku-20240307-v1:0
          aws_region_name: us-east-1
    general_settings:
      master_key: "os.environ/LITELLM_MASTER_KEY"

openwebui:
  ollama:
    enabled: false
  websocket:
    redis:
      enabled: false
    url: redis://inferencehub-redis-openwebui:6379/0
  openaiBaseApiUrl: http://inferencehub-litellm:4000/v1
  extraEnvVars:
    - name: DATABASE_URL
      valueFrom:
        secretKeyRef:
          name: inferencehub-postgresql-secret
          key: openwebui-database-url
    - name: OPENAI_API_KEY
      valueFrom:
        secretKeyRef:
          name: inferencehub-litellm-secret
          key: master-key

postgresql:
  auth:
    password: "your-postgres-password"
```

## ArgoCD Application

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: inferencehub
  namespace: argocd
spec:
  project: default
  source:
    repoURL: ghcr.io/vinay-venkatesh
    chart: inferencehub
    targetRevision: 0.2.1
    helm:
      valueFiles:
        - values.yaml
  destination:
    server: https://kubernetes.default.svc
    namespace: inferencehub
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

## Flux HelmRelease

```yaml
apiVersion: source.toolkit.fluxcd.io/v1beta2
kind: HelmRepository
metadata:
  name: inferencehub
  namespace: flux-system
spec:
  type: oci
  interval: 12h
  url: oci://ghcr.io/vinay-venkatesh
---
apiVersion: helm.toolkit.fluxcd.io/v2beta1
kind: HelmRelease
metadata:
  name: inferencehub
  namespace: inferencehub
spec:
  interval: 1h
  chart:
    spec:
      chart: inferencehub
      version: 0.2.1
      sourceRef:
        kind: HelmRepository
        name: inferencehub
        namespace: flux-system
  values:
    networking:
      gatewayAPI:
        hostname: inference.example.com
        gatewayRef:
          name: inferencehub-gateway
          namespace: envoy-gateway-system
        tls:
          issuerRef: letsencrypt-prod
    # ... remaining values as shown in the example above
```

## Upgrading

To upgrade to a new chart version:

```bash
helm upgrade inferencehub oci://ghcr.io/vinay-venkatesh/inferencehub \
  --version 0.2.1 \
  --namespace inferencehub \
  -f values.yaml
```

Check the [CHANGELOG](../CHANGELOG.md) before upgrading. Breaking changes (if any) are listed there along with migration steps.
