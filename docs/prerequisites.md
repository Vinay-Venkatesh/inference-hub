# Prerequisites

InferenceHub provisioner requires several cluster-wide components that are installed once per cluster. These are separate from the application itself because they are shared across namespaces and must exist before the Helm chart is installed.

## What gets installed

| Component | Purpose | Namespace |
|-----------|---------|-----------|
| Gateway API CRDs | Kubernetes networking primitives | cluster-scoped |
| cert-manager | Automatic TLS certificate provisioning | `cert-manager` |
| Let's Encrypt ClusterIssuer | Certificate issuer configuration | cluster-scoped |
| Envoy Gateway | Implements the Gateway API | `envoy-gateway-system` |
| GatewayClass | Declares Envoy as the gateway controller | cluster-scoped |
| Gateway | The actual load balancer / entry point | `envoy-gateway-system` |
| AWS LB Controller | Creates NLB for the Gateway (AWS only) | `kube-system` |

## Setup script

```bash
./scripts/setup-prerequisites.py [OPTIONS]
```

### Required flags

| Flag | Description |
|------|-------------|
| `--cluster-name <name>` | Name of your EKS/GKE/AKS cluster |
| `--domain <domain>` | Domain where InferenceHub will be served (e.g., `inferencehub.ai`) |

### Optional flags

| Flag | Default | Description |
|------|---------|-------------|
| `--environment <env>` | `staging` | `production` or `prod` for Let's Encrypt prod issuer; anything else uses staging |
| `--tls-email <email>` | — | Email for Let's Encrypt certificate notifications |
| `--aws-lb-role-arn <arn>` | — | IAM role ARN for AWS Load Balancer Controller (AWS only) |
| `--gateway-name <name>` | `inferencehub-gateway` | Name of the Gateway resource |
| `--gateway-namespace <ns>` | `envoy-gateway-system` | Namespace for the Gateway resource |
| `--dry-run` | — | Print what would be installed without making changes |
| `--skip-gateway-crds` | — | Skip Gateway API CRD installation |
| `--skip-cert-manager` | — | Skip cert-manager installation |
| `--skip-cluster-issuer` | — | Skip ClusterIssuer creation |
| `--skip-envoy-gateway` | — | Skip Envoy Gateway installation |
| `--skip-gateway-resources` | — | Skip GatewayClass/Gateway/EnvoyProxy creation |
| `--skip-aws-lb-controller` | — | Skip AWS Load Balancer Controller installation |

### Examples

**Staging environment (Let's Encrypt test certs):**

```bash
./scripts/setup-prerequisites.py \
  --cluster-name my-cluster \
  --domain inferencehub.ai \
  --environment staging \
  --tls-email admin@inferencehub.ai
```

**Production environment:**

```bash
./scripts/setup-prerequisites.py \
  --cluster-name my-cluster \
  --domain inferencehub.ai \
  --environment production \
  --tls-email admin@inferencehub.ai
```

**AWS EKS with Network Load Balancer:**

```bash
./scripts/setup-prerequisites.py \
  --cluster-name my-cluster \
  --domain inferencehub.ai \
  --environment staging \
  --tls-email admin@inferencehub.ai \
  --aws-lb-role-arn arn:aws:iam::123456789012:role/aws-load-balancer-controller
```

## Versions installed

| Component | Version |
|-----------|---------|
| Gateway API CRDs | v1.2.1 |
| cert-manager | v1.16.2 |
| Envoy Gateway | v1.2.1 |
| AWS LB Controller | v1.11.0 (if requested) |

## Staging vs production TLS

The `--environment` flag determines which Let's Encrypt issuer is used:

- **Staging** (default): Uses `letsencrypt-staging`. Certificates are valid but not trusted by browsers. Use this for testing.
- **Production** (`--environment production` or `--environment prod`): Uses `letsencrypt-prod`. Certificates are fully trusted. Rate limits apply — do not use for testing.

The `config.yaml` `environment` field must match what you set here. InferenceHub's `IssuerType()` method derives the ClusterIssuer name from this field.

## AWS Load Balancer Controller

Required for AWS EKS to provision a Network Load Balancer (NLB) for the Envoy Gateway.

### Create the IAM role

The controller needs an IAM role with permissions to manage AWS load balancers. Create it using `eksctl`:

```bash
eksctl create iamserviceaccount \
  --cluster=<cluster-name> \
  --namespace=kube-system \
  --name=aws-load-balancer-controller \
  --role-name=aws-load-balancer-controller \
  --attach-policy-arn=arn:aws:iam::aws:policy/ElasticLoadBalancingFullAccess \
  --approve
```

Then pass the role ARN to the script:

```bash
./scripts/setup-prerequisites.py \
  --aws-lb-role-arn arn:aws:iam::123456789012:role/aws-load-balancer-controller \
  ...
```

### Configure InferenceHub for AWS

Set `cloudProvider: aws` and `aws.litellmRoleArn` in `inferencehub.yaml`. The CLI handles everything else automatically.

```yaml
# inferencehub.yaml
cloudProvider: aws

aws:
  litellmRoleArn: "arn:aws:iam::123456789012:role/litellm-bedrock-role"
```

Then install with just:

```bash
inferencehub install --config inferencehub.yaml
```

> **Note on `values-aws.yaml`**: This file is for AWS-specific Helm overrides that apply to your cluster environment (e.g., `gp3` storage classes, AWS-specific node selectors, or specialized service annotations). It is auto-selected by the CLI. **Do not edit this file to add credentials or ARNs**; those belong in your instance-specific `inferencehub.yaml`.

## Verifying prerequisites

After running the script, check that everything is ready:

```bash
inferencehub verify
```

## Cleanup

To remove all prerequisites:

```bash
./scripts/uninstall-prerequisites.py --confirm
```

This removes (in reverse order): Gateway/GatewayClass/EnvoyProxy, Envoy Gateway, ClusterIssuer, cert-manager, Gateway API CRDs.
