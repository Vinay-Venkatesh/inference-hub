# InferenceHub Infrastructure Model

InferenceHub is an infrastructure layer provisioner that helps you to create and manage your Internal AI platform via cli.

When you run `inferencehub install`, you get a complete, production-ready AI inference stack without manually wiring together databases, caches, or LLM gateways. Every component is pre-configured to talk to every other component out of the box.

---

## What InferenceHub Provides

InferenceHub deploys and wires together four infrastructure components automatically:

| Component  | Purpose                              | In-cluster default | External option        |
|------------|--------------------------------------|-------------------|------------------------|
| PostgreSQL | Persistent storage                   | One pod, two databases | RDS, Cloud SQL, etc. |
| Redis (OpenWebUI) | Websocket session state       | One pod           | ElastiCache, Upstash, etc. |
| Redis (LiteLLM) | API response caching            | One pod           | ElastiCache, Upstash, etc. |
| LiteLLM    | LLM gateway / model router           | Managed subchart  | N/A (always deployed)  |
| OpenWebUI  | Chat interface                       | Managed subchart  | N/A (always deployed)  |
| SearXNG    | Web search engine                    | One pod           | Brave, Bing, Tavily, etc. |

---

## Datastores

The in-cluster PostgreSQL and Redis instances are managed defaults — they work out of the box with zero configuration. You are always free to swap any of them for an external managed service (RDS, ElastiCache, Cloud SQL, etc.) by pointing InferenceHub at your own connection strings. InferenceHub does not lock you in to its defaults.

This means:

- **Getting started**: use the in-cluster defaults. No database setup, no Redis configuration, no connection strings to manage.
- **Going to production**: replace any in-cluster component with a managed service by adding a few lines to `inferencehub.yaml`. The rest of the wiring stays the same.
- **Full control**: opt out of all in-cluster datastores and point every app at its own external service independently.

The sections below explain the design choices behind each datastore and how to configure them.

---

## PostgreSQL: One Cluster, Two Databases

InferenceHub deploys a single PostgreSQL pod but creates two databases: `openwebui` and `litellm`. Each application gets its own connection string and its own isolated schema. This is the standard cloud-native practice — it mirrors how a managed database like Amazon RDS is used in production, where you pay for one instance and create multiple databases on it.

**Why not two separate pods?**

- **Cost**: One RDS instance is one bill. Two instances double the cost with no benefit.
- **Ops overhead**: One set of backups, one restore path, one set of maintenance windows.
- **Isolation**: Two databases on the same PostgreSQL server are fully isolated at the schema level. A query in the LiteLLM database cannot affect the OpenWebUI database.

**When to use external PostgreSQL:**

Any production deployment where data durability and automated backups matter. Set `postgresql.openwebuiConnectionString` and `postgresql.litellmConnectionString` in `inferencehub.yaml` to point each app at its own database on your managed instance.

```yaml
postgresql:
  openwebuiConnectionString: "postgresql://user:pass@mydb.us-east-1.rds.amazonaws.com:5432/openwebui"
  litellmConnectionString:   "postgresql://user:pass@mydb.us-east-1.rds.amazonaws.com:5432/litellm"
```

---

## Redis: Two Separate Instances

Unlike PostgreSQL, InferenceHub deploys **two separate Redis pods** — one for OpenWebUI, one for LiteLLM. This is intentional.

### Why separate Redis?

Redis has a single global eviction policy that applies to all keys. The two applications have fundamentally incompatible requirements:

| App       | Redis Use Case           | Required Eviction Policy         |
|-----------|--------------------------|----------------------------------|
| OpenWebUI | Websocket session state  | `noeviction` or `volatile-lru`   |
| LiteLLM   | API response caching     | `allkeys-lru`                    |

**OpenWebUI** stores active user sessions in Redis. If Redis evicts a session key, that user is silently disconnected mid-conversation. The correct policy is `noeviction` (never evict anything) or `volatile-lru` (only evict keys that have an explicit TTL set).

**LiteLLM** stores cached LLM API responses in Redis. This is pure cache — when Redis is full, old entries should be evicted to make room for new ones. The correct policy is `allkeys-lru` (evict the least-recently-used key from all keys, regardless of TTL).

Sharing a single Redis forces a compromise that works well for neither app. A misconfigured shared Redis in production can cause:

- **User session loss**: active users get disconnected as LiteLLM fills the cache
- **Cache thrash**: LiteLLM's cached responses are evicted to protect OpenWebUI sessions, eliminating the latency benefit of caching

Separate Redis instances eliminate this class of problem entirely.

### When to use external Redis

Each app can independently point to a separate external Redis. This is recommended for production, especially on AWS where ElastiCache provides automatic failover and Multi-AZ replication.

```yaml
redis:
  openwebui:
    url: "redis://openwebui-cache.abc123.cache.amazonaws.com:6379"
    password: "${OPENWEBUI_REDIS_PASSWORD}"
  litellm:
    url: "redis://litellm-cache.def456.cache.amazonaws.com:6379"
    password: "${LITELLM_REDIS_PASSWORD}"
```

You can also mix in-cluster and external — for example, use external Redis for LiteLLM (where caching latency matters most) while keeping in-cluster Redis for OpenWebUI:

```yaml
redis:
  openwebui:
    # in-cluster (default — no config needed)
  litellm:
    url: "redis://litellm-cache.def456.cache.amazonaws.com:6379"
    password: "${LITELLM_REDIS_PASSWORD}"
```

---

## Customising beyond the defaults

For deeper customisation — custom resource limits, Redis clustering modes, PostgreSQL replication, SSO, pipelines, alerting — use the `openwebui:` and `litellm:` passthrough blocks in `inferencehub.yaml`. These accept the full upstream chart values without needing to manage the subcharts directly.

See the [configuration reference](configuration.md) for the full passthrough schema and protected key list.
