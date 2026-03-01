#!/usr/bin/env python3
"""
InferenceHub - Prerequisites Uninstall

Removes all cluster-wide prerequisites installed by setup-prerequisites.py.
Run AFTER uninstalling InferenceHub itself (inferencehub uninstall --confirm).

Usage:
  python3 scripts/uninstall-prerequisites.py [OPTIONS]
"""

import argparse
import subprocess
import sys

# ── ANSI colours ──────────────────────────────────────────────────────────────

RED    = "\033[0;31m"
GREEN  = "\033[0;32m"
YELLOW = "\033[1;33m"
BOLD   = "\033[1m"
NC     = "\033[0m"

def log_info(msg):  print(f"\033[0;34m[INFO]{NC}  {msg}")
def log_ok(msg):    print(f"{GREEN}[OK]{NC}    {msg}")
def log_warn(msg):  print(f"{YELLOW}[WARN]{NC}  {msg}")
def log_step(msg):  print(f"\n{BOLD}==> {msg}{NC}")


# ── Helpers ───────────────────────────────────────────────────────────────────

def run(cmd: list, check: bool = False):
    """Run a command, silently ignoring failures by default (safe for delete ops)."""
    subprocess.run(cmd, capture_output=True, check=check)


def kubectl_delete(*args):
    """kubectl delete with --ignore-not-found so missing resources are not errors."""
    run(["kubectl", "delete", *args, "--ignore-not-found=true"])


def helm_uninstall(release: str, namespace: str):
    """helm uninstall, ignoring errors if the release doesn't exist."""
    run(["helm", "uninstall", release, "-n", namespace, "--ignore-not-found"])


# ── Confirmation ──────────────────────────────────────────────────────────────

def confirm_uninstall(skip: bool):
    if skip:
        return
    print(f"""
{RED}WARNING: This will remove cluster-wide components.{NC}
Other workloads relying on cert-manager or Envoy Gateway will be affected.
""")
    response = input("Type 'yes' to confirm: ").strip()
    if response != "yes":
        print("Aborted.")
        sys.exit(0)


# ── Removal steps ─────────────────────────────────────────────────────────────

def remove_gateway_resources(gateway_name: str, gateway_namespace: str):
    log_step("Removing Gateway resources")
    kubectl_delete("gateway",     gateway_name,             "-n", gateway_namespace)
    kubectl_delete("envoyproxy",  f"{gateway_name}-config", "-n", gateway_namespace)
    kubectl_delete("gatewayclass", "envoy")
    log_ok("Gateway resources removed")


def remove_envoy_gateway(skip: bool, gateway_namespace: str):
    if skip:
        log_warn("Skipping Envoy Gateway removal (--skip-envoy-gateway)")
        return
    log_step("Uninstalling Envoy Gateway")
    helm_uninstall("eg", gateway_namespace)
    kubectl_delete("namespace", gateway_namespace)
    log_ok("Envoy Gateway removed")


def remove_cert_manager(skip: bool):
    if skip:
        log_warn("Skipping cert-manager removal (--skip-cert-manager)")
        return
    log_step("Uninstalling cert-manager")
    helm_uninstall("cert-manager", "cert-manager")
    kubectl_delete("namespace", "cert-manager")
    log_ok("cert-manager removed")


def remove_aws_lb_controller(skip: bool):
    if skip:
        log_warn("Skipping AWS Load Balancer Controller removal (--skip-aws-lb-controller)")
        return
    log_step("Uninstalling AWS Load Balancer Controller")
    helm_uninstall("aws-load-balancer-controller", "kube-system")
    log_ok("AWS Load Balancer Controller removed")


# ── Argument parsing ──────────────────────────────────────────────────────────

def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        prog="uninstall-prerequisites.py",
        description="Remove cluster-wide InferenceHub prerequisites.",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""\
warning:
  This removes cluster-wide components (cert-manager, Envoy Gateway, etc.)
  that may be shared by other workloads. Use --skip-* flags to preserve
  components still needed by other applications.

examples:
  # Remove everything
  python3 scripts/uninstall-prerequisites.py --confirm

  # Keep cert-manager (used by other apps)
  python3 scripts/uninstall-prerequisites.py --confirm --skip-cert-manager

  # Keep AWS LB Controller (shared across namespaces)
  python3 scripts/uninstall-prerequisites.py --confirm --skip-aws-lb-controller
""",
    )
    parser.add_argument("--gateway-name",           default="inferencehub-gateway", help="Gateway resource name  (default: inferencehub-gateway)")
    parser.add_argument("--gateway-namespace",      default="envoy-gateway-system", help="Gateway namespace  (default: envoy-gateway-system)")
    parser.add_argument("--skip-cert-manager",      action="store_true",            help="Skip cert-manager removal")
    parser.add_argument("--skip-envoy-gateway",     action="store_true",            help="Skip Envoy Gateway removal")
    parser.add_argument("--skip-aws-lb-controller", action="store_true",            help="Skip AWS Load Balancer Controller removal")
    parser.add_argument("--confirm",                action="store_true",            help="Skip confirmation prompt")
    return parser.parse_args()


# ── Entry point ───────────────────────────────────────────────────────────────

def main():
    args = parse_args()

    print(f"""
{BOLD}============================================================{NC}
{BOLD}InferenceHub Prerequisites Uninstall{NC}
{BOLD}============================================================{NC}

  Gateway:  {args.gateway_name} / {args.gateway_namespace}
""")

    confirm_uninstall(args.confirm)

    remove_gateway_resources(args.gateway_name, args.gateway_namespace)
    remove_envoy_gateway(args.skip_envoy_gateway, args.gateway_namespace)
    remove_cert_manager(args.skip_cert_manager)
    remove_aws_lb_controller(args.skip_aws_lb_controller)

    print(f"\n{GREEN}Prerequisites uninstalled.{NC}\n")


if __name__ == "__main__":
    main()
