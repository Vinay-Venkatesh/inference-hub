#!/usr/bin/env python3
"""
InferenceHub - Prerequisites Setup

Installs all cluster-wide prerequisites required by InferenceHub:
  1. Gateway API CRDs
  2. cert-manager
  3. ClusterIssuer (Let's Encrypt)
  4. Envoy Gateway
  5. GatewayClass + EnvoyProxy + Gateway + ReferenceGrant
  6. AWS Load Balancer Controller (optional, AWS EKS only)

Usage:
  python3 scripts/setup-prerequisites.py --cluster-name <name> --domain <fqdn> [OPTIONS]
"""

import argparse
import json
import os
import re
import subprocess
import sys
import tempfile
from typing import Optional

# ── Pinned versions ───────────────────────────────────────────────────────────

GATEWAY_API_VERSION       = "v1.5.0"
CERT_MANAGER_VERSION      = "v1.19.4"
ENVOY_GATEWAY_VERSION     = "v1.7.0"
AWS_LB_CONTROLLER_VERSION = "3.1.0"

# ── ANSI colours ──────────────────────────────────────────────────────────────

RED    = "\033[0;31m"
GREEN  = "\033[0;32m"
YELLOW = "\033[1;33m"
BLUE   = "\033[0;34m"
BOLD   = "\033[1m"
NC     = "\033[0m"

def log_info(msg):  print(f"{BLUE}[INFO]{NC}  {msg}")
def log_ok(msg):    print(f"{GREEN}[OK]{NC}    {msg}")
def log_warn(msg):  print(f"{YELLOW}[WARN]{NC}  {msg}")
def log_error(msg): print(f"{RED}[ERROR]{NC} {msg}", file=sys.stderr)
def log_step(msg):  print(f"\n{BOLD}==> {msg}{NC}")

# ── Command runner ────────────────────────────────────────────────────────────

_dry_run = False


def run(cmd: list, stdin: Optional[str] = None, check: bool = True) -> subprocess.CompletedProcess:
    """Execute a command, or print it when --dry-run is set."""
    if _dry_run:
        display = " ".join(str(a) for a in cmd)
        if stdin:
            display += f" <<EOF\n{stdin.strip()}\nEOF"
        print(f"{YELLOW}[DRY-RUN]{NC} {display}")
        return subprocess.CompletedProcess(cmd, 0, stdout="", stderr="")
    return subprocess.run(cmd, input=stdin, text=True, check=check)


def capture(cmd: list) -> str:
    """Run a read-only command and return stdout (ignores dry-run)."""
    result = subprocess.run(cmd, capture_output=True, text=True)
    return result.stdout.strip()


# ── Preflight ─────────────────────────────────────────────────────────────────

def check_command(name: str):
    if subprocess.run(["which", name], capture_output=True).returncode != 0:
        log_error(f"Required command '{name}' not found. Please install it first.")
        sys.exit(1)


# ── Idempotency helpers ───────────────────────────────────────────────────────

def helm_release_exists(release: str, namespace: str) -> Optional[str]:
    """Return the chart string (e.g. 'cert-manager-v1.19.4') if installed, else None."""
    out = capture(["helm", "list", "-n", namespace, "--filter", f"^{release}$", "-o", "json"])
    try:
        releases = json.loads(out)
        if releases:
            return releases[0].get("chart", "unknown")
    except (json.JSONDecodeError, IndexError):
        pass
    return None


def resource_exists(kind: str, name: str, namespace: Optional[str] = None) -> bool:
    cmd = ["kubectl", "get", kind, name]
    if namespace:
        cmd += [f"--namespace={namespace}"]
    return subprocess.run(cmd, capture_output=True).returncode == 0


# ── ACME helpers ──────────────────────────────────────────────────────────────

def issuer_name(environment: str) -> str:
    return "letsencrypt-prod" if environment in ("production", "prod") else "letsencrypt-staging"


def acme_server_url(environment: str) -> str:
    return (
        "https://acme-v02.api.letsencrypt.org/directory"
        if environment in ("production", "prod")
        else "https://acme-staging-v02.api.letsencrypt.org/directory"
    )


# ── Step 1: Gateway API CRDs ──────────────────────────────────────────────────

def install_gateway_api_crds():
    log_step(f"Step 1/6: Gateway API CRDs ({GATEWAY_API_VERSION})")

    if resource_exists("crd", "gateways.gateway.networking.k8s.io"):
        version = capture([
            "kubectl", "get", "crd", "gateways.gateway.networking.k8s.io",
            "-o", r"jsonpath={.metadata.annotations.gateway\.networking\.k8s\.io/bundle-version}",
        ]) or "unknown"
        log_ok(f"Gateway API CRDs already installed ({version})")
        return

    log_info(f"Installing Gateway API CRDs {GATEWAY_API_VERSION}...")
    url = f"https://github.com/kubernetes-sigs/gateway-api/releases/download/{GATEWAY_API_VERSION}/standard-install.yaml"
    run(["kubectl", "apply", "-f", url])
    log_ok("Gateway API CRDs installed")


# ── Step 2: cert-manager ──────────────────────────────────────────────────────

def install_cert_manager(skip: bool):
    log_step(f"Step 2/6: cert-manager ({CERT_MANAGER_VERSION})")

    if skip:
        log_warn("Skipping cert-manager installation (--skip-cert-manager)")
        return

    chart = helm_release_exists("cert-manager", "cert-manager")
    if chart:
        log_ok(f"cert-manager already installed ({chart})")
        return

    log_info("Adding jetstack Helm repository...")
    run(["helm", "repo", "add", "jetstack", "https://charts.jetstack.io", "--force-update"])
    run(["helm", "repo", "update", "jetstack"])

    log_info(f"Installing cert-manager {CERT_MANAGER_VERSION}...")
    run([
        "helm", "install", "cert-manager", "jetstack/cert-manager",
        "--namespace", "cert-manager",
        "--create-namespace",
        "--version", CERT_MANAGER_VERSION,
        "--set", "crds.enabled=true",
        "--set", "crds.keep=true",
        "--set", "extraArgs[0]=--enable-gateway-api",
        "--wait",
        "--timeout", "5m",
    ])
    log_ok("cert-manager installed")


# ── Step 3: ClusterIssuer ─────────────────────────────────────────────────────

def install_cluster_issuer(environment: str, tls_email: str, gateway_name: str, gateway_namespace: str):
    log_step("Step 3/6: ClusterIssuer (Let's Encrypt)")

    issuer = issuer_name(environment)
    server = acme_server_url(environment)

    if resource_exists("clusterissuer", issuer):
        log_ok(f"ClusterIssuer '{issuer}' already exists")
        return

    if not tls_email:
        log_warn("No --tls-email provided. Skipping ClusterIssuer creation.")
        log_warn("Create it manually or re-run with --tls-email <email>")
        return

    log_info(f"Creating ClusterIssuer '{issuer}' for environment '{environment}'...")
    run(["kubectl", "apply", "-f", "-"], stdin=f"""\
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: {issuer}
spec:
  acme:
    server: {server}
    email: {tls_email}
    privateKeySecretRef:
      name: {issuer}-key
    solvers:
      - http01:
          gatewayHTTPRoute:
            parentRefs:
              - name: {gateway_name}
                namespace: {gateway_namespace}
                kind: Gateway
""")
    log_ok(f"ClusterIssuer '{issuer}' created")

    if environment not in ("production", "prod"):
        log_info(f"Using staging issuer for '{environment}'. Re-run with --environment production for trusted certs.")


# ── Step 4: Envoy Gateway ─────────────────────────────────────────────────────

def install_envoy_gateway(skip: bool, gateway_namespace: str):
    log_step(f"Step 4/6: Envoy Gateway ({ENVOY_GATEWAY_VERSION})")

    if skip:
        log_warn("Skipping Envoy Gateway installation (--skip-envoy-gateway)")
        return

    chart = helm_release_exists("eg", gateway_namespace)
    if chart:
        log_ok(f"Envoy Gateway already installed ({chart})")
        return

    # The Envoy Gateway chart bundles two sets of CRDs:
    #   - gateway.networking.k8s.io  (standard + experimental Gateway API CRDs)
    #   - gateway.envoyproxy.io      (Envoy Gateway's own CRDs)
    #
    # The Gateway API CRDs were already installed in Step 1. Re-applying them,
    # especially the experimental ones (TCPRoute, UDPRoute, TLSRoute), triggers
    # the safe-upgrades ValidatingAdmissionPolicy and causes an error.
    #
    # We fetch all CRDs from the chart, filter out any whose metadata.name falls
    # in the gateway.networking.k8s.io group, and apply only the remainder.
    # The check is on the name: line specifically — Envoy Gateway's own CRDs
    # reference gateway.networking.k8s.io types deep in their OpenAPI schemas,
    # so a full-document string match would incorrectly exclude them.
    log_info("Fetching Envoy Gateway CRDs from chart...")
    if _dry_run:
        print(f"{YELLOW}[DRY-RUN]{NC} helm show crds oci://docker.io/envoyproxy/gateway-helm:{ENVOY_GATEWAY_VERSION} "
              f"| filter *.gateway.networking.k8s.io | kubectl apply --server-side --force-conflicts -f -")
    else:
        result = subprocess.run(
            ["helm", "show", "crds", "oci://docker.io/envoyproxy/gateway-helm", "--version", ENVOY_GATEWAY_VERSION],
            capture_output=True, text=True, check=True,
        )
        docs = result.stdout.split("\n---")
        envoy_crds = [
            d.strip() for d in docs
            if d.strip() and not re.search(
                r"^\s+name:\s+\S+\.gateway\.networking\.k8s\.io\s*$", d, re.MULTILINE
            )
        ]
        if envoy_crds:
            log_info(f"Applying {len(envoy_crds)} Envoy Gateway-specific CRDs (server-side, force-conflicts)...")
            subprocess.run(
                ["kubectl", "apply", "--server-side", "--force-conflicts", "-f", "-"],
                input="\n---\n".join(envoy_crds),
                text=True, check=True,
            )
        else:
            log_warn("No Envoy Gateway-specific CRDs found after filtering — may already be current")

    log_info(f"Installing Envoy Gateway {ENVOY_GATEWAY_VERSION}...")
    run([
        "helm", "install", "eg", "oci://docker.io/envoyproxy/gateway-helm",
        "--version", ENVOY_GATEWAY_VERSION,
        "--namespace", gateway_namespace,
        "--create-namespace",
        "--skip-crds",
        "--wait",
        "--timeout", "5m",
    ])
    log_ok("Envoy Gateway installed")


# ── Step 5: GatewayClass + EnvoyProxy + Gateway + ReferenceGrant ─────────────

def install_gateway_resources(gateway_name: str, gateway_namespace: str, platform_namespace: str):
    log_step("Step 5/6: GatewayClass + Gateway + EnvoyProxy")

    # GatewayClass
    if resource_exists("gatewayclass", "envoy"):
        log_ok("GatewayClass 'envoy' already exists")
    else:
        log_info("Creating GatewayClass 'envoy'...")
        run(["kubectl", "apply", "-f", "-"], stdin=f"""\
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: envoy
spec:
  controllerName: gateway.envoyproxy.io/gatewayclass-controller
  parametersRef:
    group: gateway.envoyproxy.io
    kind: EnvoyProxy
    name: {gateway_name}-config
    namespace: {gateway_namespace}
""")
        log_ok("GatewayClass 'envoy' created")

    # EnvoyProxy — AWS NLB configuration
    if resource_exists("envoyproxy", f"{gateway_name}-config", gateway_namespace):
        log_ok(f"EnvoyProxy '{gateway_name}-config' already exists")
    else:
        log_info(f"Creating EnvoyProxy '{gateway_name}-config' (AWS NLB)...")
        run(["kubectl", "apply", "-f", "-"], stdin=f"""\
apiVersion: gateway.envoyproxy.io/v1alpha1
kind: EnvoyProxy
metadata:
  name: {gateway_name}-config
  namespace: {gateway_namespace}
spec:
  provider:
    type: Kubernetes
    kubernetes:
      envoyService:
        type: LoadBalancer
        annotations:
          service.beta.kubernetes.io/aws-load-balancer-type: external
          service.beta.kubernetes.io/aws-load-balancer-nlb-target-type: ip
          service.beta.kubernetes.io/aws-load-balancer-scheme: internet-facing
          service.beta.kubernetes.io/aws-load-balancer-cross-zone-load-balancing-enabled: "true"
        externalTrafficPolicy: Local
      envoyDeployment:
        container:
          resources:
            requests:
              cpu: 100m
              memory: 256Mi
            limits:
              cpu: 500m
              memory: 512Mi
""")
        log_ok("EnvoyProxy created")

    # Gateway
    if resource_exists("gateway", gateway_name, gateway_namespace):
        log_ok(f"Gateway '{gateway_name}' already exists")
    else:
        log_info(f"Creating Gateway '{gateway_name}'...")
        run(["kubectl", "apply", "-f", "-"], stdin=f"""\
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: {gateway_name}
  namespace: {gateway_namespace}
spec:
  gatewayClassName: envoy
  listeners:
    - name: http
      protocol: HTTP
      port: 80
      allowedRoutes:
        namespaces:
          from: All
    - name: https
      protocol: HTTPS
      port: 443
      tls:
        mode: Terminate
        certificateRefs:
          - kind: Secret
            name: inferencehub-tls
            namespace: {platform_namespace}
      allowedRoutes:
        namespaces:
          from: All
""")
        log_ok(f"Gateway '{gateway_name}' created")

    # ReferenceGrant — permits the Gateway in gateway_namespace to reference the
    # TLS Secret in platform_namespace. Without this the HTTPS listener reports
    # RefNotPermitted and the Gateway is never Programmed.
    if resource_exists("referencegrant", "allow-gateway-tls", platform_namespace):
        log_ok(f"ReferenceGrant 'allow-gateway-tls' already exists in {platform_namespace}")
    else:
        log_info(f"Ensuring namespace '{platform_namespace}' exists...")
        if not _dry_run:
            # create namespace idempotently
            subprocess.run(
                ["kubectl", "create", "namespace", platform_namespace],
                capture_output=True,  # suppress "already exists" noise
            )

        log_info("Creating ReferenceGrant (Gateway → Secret cross-namespace TLS)...")
        run(["kubectl", "apply", "-f", "-"], stdin=f"""\
apiVersion: gateway.networking.k8s.io/v1beta1
kind: ReferenceGrant
metadata:
  name: allow-gateway-tls
  namespace: {platform_namespace}
spec:
  from:
    - group: gateway.networking.k8s.io
      kind: Gateway
      namespace: {gateway_namespace}
  to:
    - group: ""
      kind: Secret
""")
        log_ok(f"ReferenceGrant created — Gateway can now reference TLS secrets in {platform_namespace}")


# ── Step 6: AWS Load Balancer Controller ──────────────────────────────────────

def install_aws_lb_controller(skip: bool, role_arn: str, cluster_name: str):
    log_step(f"Step 6/6: AWS Load Balancer Controller ({AWS_LB_CONTROLLER_VERSION})")

    if skip:
        log_warn("Skipping AWS Load Balancer Controller (--skip-aws-lb-controller)")
        return

    if not role_arn:
        log_warn("No --aws-lb-role-arn provided. Skipping AWS Load Balancer Controller.")
        log_warn("Re-run with --aws-lb-role-arn <arn> to install it.")
        return

    chart = helm_release_exists("aws-load-balancer-controller", "kube-system")
    if chart:
        log_ok(f"AWS Load Balancer Controller already installed ({chart})")
        return

    log_info("Adding eks Helm repository...")
    run(["helm", "repo", "add", "eks", "https://aws.github.io/eks-charts", "--force-update"])
    run(["helm", "repo", "update", "eks"])

    # Write IRSA annotation to a temp file — the annotation key contains dots and a
    # slash that are unreliable to pass via --set flags.
    values_yaml = f"""\
clusterName: "{cluster_name}"
serviceAccount:
  create: true
  name: aws-load-balancer-controller
  annotations:
    eks.amazonaws.com/role-arn: "{role_arn}"
"""
    with tempfile.NamedTemporaryFile(mode="w", suffix=".yaml", prefix="alb-values-", delete=False) as f:
        f.write(values_yaml)
        tmp_path = f.name

    try:
        log_info(f"Installing AWS Load Balancer Controller {AWS_LB_CONTROLLER_VERSION}...")
        run([
            "helm", "install", "aws-load-balancer-controller", "eks/aws-load-balancer-controller",
            "--namespace", "kube-system",
            "--version", AWS_LB_CONTROLLER_VERSION,
            "-f", tmp_path,
            "--wait",
            "--timeout", "5m",
        ])
        log_ok("AWS Load Balancer Controller installed")
    finally:
        os.unlink(tmp_path)


# ── Summary ───────────────────────────────────────────────────────────────────

def print_summary(domain: str, environment: str, gateway_name: str, gateway_namespace: str):
    issuer = issuer_name(environment)
    print(f"""
{BOLD}============================================================{NC}
{BOLD}Prerequisites Setup Complete{NC}
{BOLD}============================================================{NC}

{GREEN}Use these values in your config.yaml:{NC}

  gateway:
    name: {gateway_name}
    namespace: {gateway_namespace}

  environment: {environment}  # issuerType: {issuer}

{BOLD}Next steps:{NC}
  1. Point your DNS: {domain} → <NLB hostname>
     kubectl get gateway {gateway_name} -n {gateway_namespace} \\
       -o jsonpath='{{.status.addresses[0].value}}'

  2. Install InferenceHub:
     inferencehub install --config inferencehub.yaml
""")


# ── Argument parsing ──────────────────────────────────────────────────────────

def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        prog="setup-prerequisites.py",
        description="Install cluster-wide prerequisites for InferenceHub on AWS EKS.",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=f"""\
version pins:
  Gateway API CRDs:        {GATEWAY_API_VERSION}
  cert-manager:            {CERT_MANAGER_VERSION}
  Envoy Gateway:           {ENVOY_GATEWAY_VERSION}
  AWS Load Balancer Ctrl:  {AWS_LB_CONTROLLER_VERSION}

examples:
  # Full AWS setup
  python3 scripts/setup-prerequisites.py \\
    --cluster-name my-cluster \\
    --domain ai.company.com \\
    --environment production \\
    --tls-email admin@company.com \\
    --aws-lb-role-arn arn:aws:iam::123456789:role/AWSLoadBalancerControllerRole

  # Skip components already installed
  python3 scripts/setup-prerequisites.py \\
    --cluster-name my-cluster \\
    --domain ai.company.com \\
    --skip-aws-lb-controller

  # Preview all changes without applying them
  python3 scripts/setup-prerequisites.py \\
    --cluster-name my-cluster --domain ai.company.com --dry-run
""",
    )
    req = parser.add_argument_group("required")
    req.add_argument("--cluster-name",           required=True,  help="EKS cluster name")
    req.add_argument("--domain",                 required=True,  help="Domain for InferenceHub (e.g. ai.company.com)")

    opt = parser.add_argument_group("optional")
    opt.add_argument("--environment",            default="staging",               help="staging | production  (default: staging)")
    opt.add_argument("--tls-email",              default="",                      help="Email for Let's Encrypt certificates")
    opt.add_argument("--aws-lb-role-arn",        default="",                      help="IAM role ARN for AWS Load Balancer Controller")
    opt.add_argument("--gateway-name",           default="inferencehub-gateway",  help="Gateway resource name  (default: inferencehub-gateway)")
    opt.add_argument("--gateway-namespace",      default="envoy-gateway-system",  help="Gateway namespace  (default: envoy-gateway-system)")
    opt.add_argument("--platform-namespace",     default="inferencehub",          help="Namespace InferenceHub will be installed into  (default: inferencehub)")
    opt.add_argument("--skip-cert-manager",      action="store_true",             help="Skip cert-manager installation")
    opt.add_argument("--skip-envoy-gateway",     action="store_true",             help="Skip Envoy Gateway installation")
    opt.add_argument("--skip-aws-lb-controller", action="store_true",             help="Skip AWS Load Balancer Controller installation")
    opt.add_argument("--dry-run",                action="store_true",             help="Print commands without executing")

    return parser.parse_args()


# ── Entry point ───────────────────────────────────────────────────────────────

def main():
    global _dry_run
    args = parse_args()
    _dry_run = args.dry_run

    dry_label = f"  {YELLOW}Mode: DRY RUN (no changes will be made){NC}\n" if _dry_run else ""
    print(f"""
{BOLD}============================================================{NC}
{BOLD}InferenceHub Prerequisites Setup{NC}
{BOLD}============================================================{NC}
{dry_label}
  Cluster:     {args.cluster_name}
  Domain:      {args.domain}
  Environment: {args.environment} (issuer: {issuer_name(args.environment)})
  Gateway:     {args.gateway_name} / {args.gateway_namespace}
""")

    check_command("kubectl")
    check_command("helm")

    install_gateway_api_crds()
    install_cert_manager(args.skip_cert_manager)
    install_cluster_issuer(args.environment, args.tls_email, args.gateway_name, args.gateway_namespace)
    install_envoy_gateway(args.skip_envoy_gateway, args.gateway_namespace)
    install_gateway_resources(args.gateway_name, args.gateway_namespace, args.platform_namespace)
    install_aws_lb_controller(args.skip_aws_lb_controller, args.aws_lb_role_arn, args.cluster_name)

    print_summary(args.domain, args.environment, args.gateway_name, args.gateway_namespace)


if __name__ == "__main__":
    main()
