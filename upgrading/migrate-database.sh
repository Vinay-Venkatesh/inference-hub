#!/usr/bin/env bash
# migrate-database.sh — v0.1.0 → v0.2.0 database migration
#
# Splits the single "inferencehub" database into two:
#   - openwebui  (OpenWebUI application tables)
#   - litellm    (LiteLLM starts empty; it self-initialises on startup)
#
# Prerequisites:
#   - psql available on PATH
#   - kubectl port-forward running: kubectl port-forward ... 5432:5432
#   - PGPASSWORD env var set to the PostgreSQL password
#
# Usage:
#   PGPASSWORD=<password> bash docs/upgrading/migrate-database.sh

set -euo pipefail

PG_HOST="${PG_HOST:-localhost}"
PG_PORT="${PG_PORT:-5432}"
PG_USER="${PG_USER:-inferencehub}"
SOURCE_DB="inferencehub"

PSQL="psql -h $PG_HOST -p $PG_PORT -U $PG_USER"

echo "==> InferenceHub v0.1→v0.2 database migration"
echo "    Host: $PG_HOST:$PG_PORT  User: $PG_USER  Source: $SOURCE_DB"
echo ""

# Verify connection
echo "--> Verifying connection to source database..."
$PSQL -d "$SOURCE_DB" -c "SELECT version();" > /dev/null
echo "    Connected."

# Create openwebui database
echo "--> Creating 'openwebui' database..."
$PSQL -d postgres -tc "SELECT 1 FROM pg_database WHERE datname = 'openwebui'" | grep -q 1 \
  || $PSQL -d postgres -c "CREATE DATABASE openwebui;"
$PSQL -d postgres -c "GRANT ALL PRIVILEGES ON DATABASE openwebui TO $PG_USER;"
echo "    Done."

# Create litellm database (empty — LiteLLM will self-initialise)
echo "--> Creating 'litellm' database..."
$PSQL -d postgres -tc "SELECT 1 FROM pg_database WHERE datname = 'litellm'" | grep -q 1 \
  || $PSQL -d postgres -c "CREATE DATABASE litellm;"
$PSQL -d postgres -c "GRANT ALL PRIVILEGES ON DATABASE litellm TO $PG_USER;"
echo "    Done (empty — LiteLLM will initialise on first startup)."

# OpenWebUI tables to migrate from inferencehub → openwebui
# These are the standard OpenWebUI tables as of v0.8.x
OW_TABLES=(
  auth
  channel
  channel_member
  chat
  config
  document
  file
  folder
  "group"
  memory
  message
  model
  prompt
  tag
  tool
  "user"
)

echo "--> Migrating OpenWebUI tables to 'openwebui' database..."
for table in "${OW_TABLES[@]}"; do
  # Check if table exists in source
  EXISTS=$($PSQL -d "$SOURCE_DB" -tc "SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name='$table'" | tr -d ' ')
  if [ "$EXISTS" = "1" ]; then
    echo "    Copying table: $table"
    $PSQL -d "$SOURCE_DB" -c "\COPY $table TO STDOUT" \
      | $PSQL -d openwebui -c "
          CREATE TABLE IF NOT EXISTS $table (LIKE $table INCLUDING ALL);
          \COPY $table FROM STDIN
        " 2>/dev/null || true
  else
    echo "    Skipping (not found): $table"
  fi
done

echo ""
echo "==> Migration complete."
echo ""
echo "Next steps:"
echo "  1. Run:  inferencehub upgrade --config inferencehub.yaml"
echo "  2. Run:  inferencehub verify"
echo "  3. Once verified, the old 'inferencehub' database can be dropped:"
echo "     PGPASSWORD=<password> psql -h $PG_HOST -p $PG_PORT -U $PG_USER -d postgres -c 'DROP DATABASE inferencehub;'"
