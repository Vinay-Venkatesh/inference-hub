# Contributing to InferenceHub

Thank you for your interest in contributing!

## Development setup

### Prerequisites

- Go 1.21+
- Helm 3.x
- kubectl
- A Kubernetes cluster (local `kind` cluster works well)

### Build the CLI

```bash
cd inferencehub-cli
go build ./...
go vet ./...
```

### Lint the Helm chart

```bash
helm lint helm/inferencehub/
helm template inferencehub helm/inferencehub/ --values helm/inferencehub/values-local.yaml
```

## Project structure

```
inference-platform/
├── helm/inferencehub/   # The Helm chart
├── scripts/             # Prerequisites and utility scripts
├── inferencehub-cli/    # CLI source code (Go + Cobra)
├── examples/            # Sample configs and .env
└── docs/                # Documentation
```

## How to contribute

1. **Fork** the repository and clone it locally.
2. **Create a branch** for your change: `git checkout -b feature/your-feature`.
3. **Make your changes.** Keep changes focused and minimal.
4. **Test:**
   - Helm changes: `helm lint` + `helm template` must pass
   - CLI changes: `go build ./...` and `go vet ./...` must pass
5. **Open a pull request** with a clear description of what and why.

## Reporting issues

Open an issue on GitHub with:
- What you were trying to do
- What happened instead
- Relevant config (redact any secrets)
- Cluster environment (EKS, kind, etc.) and Kubernetes version

## Code style

- Go: standard `gofmt` formatting
- Helm: follow [Helm chart best practices](https://helm.sh/docs/chart_best_practices/)
- No hardcoded secrets, ARNs, or personal information in committed files
- Prefer editing existing files over creating new ones
