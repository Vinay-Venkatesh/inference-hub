# InferenceHub CLI

A command-line tool for installing, managing, and verifying the InferenceHub platform on Kubernetes.

## Installation

**Prerequisites:** Go 1.21+, `helm` v3, `kubectl`

### Option 1 — Install to GOPATH/bin (recommended)

```bash
cd inferencehub-cli
make install
```

Then add GOPATH/bin to your PATH if it isn't already (add to `~/.zshrc` or `~/.bashrc`):

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
source ~/.zshrc   # or source ~/.bashrc
```

Verify:

```bash
inferencehub --help
```

### Option 2 — Install system-wide (no PATH change needed)

```bash
cd inferencehub-cli
make install-system   # requires sudo
```

```bash
inferencehub --help
```

### Option 3 — Build only (run from repo)

```bash
cd inferencehub-cli
make build
./bin/inferencehub --help
```

## Working directory

> **Important:** Always run `inferencehub` commands from the **project root** (the directory that contains `helm/`, `scripts/`, and your `inferencehub.yaml`), not from inside `inferencehub-cli/`.
>
> The CLI creates `inferencehub.yaml` in your current directory and loads `.env` / `.env.local` from there too. Running from the project root keeps all config and secret files in one place and lets the CLI auto-detect the Helm chart at `helm/inferencehub/`.

```bash
# After installing, go to the project root before using the CLI
cd /path/to/inference-platform   # or wherever your helm/ and scripts/ live
inferencehub config init         # creates inferencehub.yaml here
inferencehub install --config inferencehub.yaml
```

## Usage

```
inferencehub [command] [flags]
```

### Global flags

| Flag | Default | Description |
|------|---------|-------------|
| `--kubeconfig` | `~/.kube/config` | Path to kubeconfig file |
| `--namespace` | `inferencehub` | Kubernetes namespace |
| `--verbose` / `-v` | `false` | Enable verbose output |

### Commands

#### `config init`

Generate a starter `inferencehub.yaml` in the current directory.

```bash
inferencehub config init
inferencehub config init --output my-config.yaml
```

#### `config validate`

Validate config file and check required environment variables.

```bash
inferencehub config validate --config inferencehub.yaml
```

#### `config show`

Show the effective configuration after environment variable interpolation.

```bash
inferencehub config show --config inferencehub.yaml
```

#### `install`

Install InferenceHub on a Kubernetes cluster.

```bash
# Minimal — values file auto-selected from cloudProvider in config
inferencehub install --config inferencehub.yaml

# Override cloud provider at runtime (takes precedence over config)
inferencehub install --config inferencehub.yaml --cloud-provider aws

# Explicit values file (highest priority, bypasses auto-selection)
inferencehub install --config inferencehub.yaml --values ./custom-overrides.yaml

# Skip confirmation prompt
inferencehub install --config inferencehub.yaml --auto-approve
```

| Flag | Required | Description |
|------|----------|-------------|
| `--config` / `-c` | Yes | Path to `config.yaml` |
| `--cloud-provider` | No | `aws`, `gcp`, `azure`, `local` — overrides `cloudProvider` in config |
| `--values` / `-f` | No | Explicit path to a Helm values file (overrides `--cloud-provider`) |
| `--chart` | No | Path to Helm chart directory (auto-detected) |
| `--auto-approve` | No | Skip confirmation prompt |

**Values file resolution order** (highest to lowest priority):
1. `--values <path>` — explicit file, always used as-is
2. `cloudProvider` from config (or `--cloud-provider` flag) — auto-resolves to `helm/inferencehub/values-{provider}.yaml`
3. No extra values file — chart defaults apply

#### `upgrade`

Apply configuration changes to an existing installation.

```bash
inferencehub upgrade --config inferencehub.yaml
```

| Flag | Required | Description |
|------|----------|-------------|
| `--config` / `-c` | Yes | Path to `config.yaml` |
| `--cloud-provider` | No | `aws`, `gcp`, `azure`, `local` — overrides `cloudProvider` in config |
| `--values` / `-f` | No | Explicit path to a Helm values file (overrides `--cloud-provider`) |
| `--auto-approve` | No | Skip confirmation prompt |

#### `status`

Show the current health of all components.

```bash
inferencehub status
inferencehub status --namespace inferencehub
```

#### `verify`

Check infrastructure prerequisites and platform pod health.

```bash
inferencehub verify
```

Reports on:
- Gateway API CRDs
- cert-manager pod health
- Envoy Gateway pod health
- InferenceHub pod readiness

#### `uninstall`

Remove the InferenceHub Helm release.

```bash
inferencehub uninstall --confirm
inferencehub uninstall --confirm --purge   # also delete PVCs
```

## Environment variables

The CLI auto-loads `.env`, `.env.local`, and `~/.inferencehub/.env` before running any command.

| Variable | Required | Description |
|----------|----------|-------------|
| `LITELLM_MASTER_KEY` | Yes | LiteLLM API key (must start with `sk-`) |
| `LANGFUSE_PUBLIC_KEY` | When observability enabled | Langfuse public key |
| `LANGFUSE_SECRET_KEY` | When observability enabled | Langfuse secret key |
| `POSTGRES_PASSWORD` | When using external PostgreSQL | Database password |
| `REDIS_PASSWORD` | When using external Redis | Cache password |

## Development

```bash
cd inferencehub-cli

# Build
make build

# Vet / format
go vet ./...
go fmt ./...

# Run a command without installing
go run ./cmd/inferencehub/main.go status --namespace inferencehub

# Run tests
make test
```
