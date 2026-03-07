# CLI Reference

All `inferencehub` commands. Run `inferencehub <command> --help` for all flags.

## Commands

| Command | Description |
|---------|-------------|
| `inferencehub config init` | Generate a starter `inferencehub.yaml` in the current directory |
| `inferencehub config validate --config <file>` | Validate config and check required env vars |
| `inferencehub config show --config <file>` | Show config after env var interpolation |
| `inferencehub install --config <file>` | Install the platform onto the cluster |
| `inferencehub upgrade --config <file>` | Upgrade an existing installation |
| `inferencehub status` | Show component health |
| `inferencehub verify` | Check prerequisites and platform readiness |
| `inferencehub uninstall --confirm` | Remove the platform |

---

## Supported model providers

| Provider | Example model ID |
|----------|-----------------|
| AWS Bedrock | `anthropic.claude-3-5-sonnet-20241022-v2:0` |
| OpenAI | `gpt-4o` |
| Ollama (self-hosted) | `llama3.2:3b` |
| Azure OpenAI | `gpt-4` |

---

## Working directory

Run all `inferencehub` commands from the **project root** — the directory that contains `helm/` and `scripts/`. The CLI reads `inferencehub.yaml` and `.env` from the current directory.

```bash
cd /path/to/inference-hub
inferencehub install --config inferencehub.yaml
```

---

## Environment variables

The CLI auto-loads env files in this order:

1. `.env` (project root)
2. `.env.local` (project root)
3. `~/.inferencehub/.env` (user global)

| Variable | Required | Description |
|----------|----------|-------------|
| `LITELLM_MASTER_KEY` | Yes | Must start with `sk-`. Used as the LiteLLM API master key. |
| `LANGFUSE_PUBLIC_KEY` | No | Required only when `observability.enabled: true` |
| `LANGFUSE_SECRET_KEY` | No | Required only when `observability.enabled: true` |
