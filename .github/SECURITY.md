# Security Policy

## Supported versions

| Version | Supported |
|---------|-----------|
| v0.1.x  | Yes       |

## Reporting a vulnerability

**Please do not open a public GitHub issue for security vulnerabilities.**

Report security issues privately via [GitHub's private vulnerability reporting](https://github.com/Vinay-Venkatesh/inference-hub/security/advisories/new).

Include:
- A description of the vulnerability and its potential impact
- Steps to reproduce
- Any suggested fix, if you have one

You can expect an acknowledgement within 48 hours and a resolution timeline within 7 days depending on severity.

## Scope

This covers:
- The `inferencehub` CLI
- The Helm chart (`helm/inferencehub/`)
- The prerequisites scripts (`scripts/`)

Out of scope: vulnerabilities in upstream dependencies (OpenWebUI, LiteLLM, cert-manager, Envoy Gateway). Please report those to their respective projects.
