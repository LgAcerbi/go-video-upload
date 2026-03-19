#!/usr/bin/env bash
set -euo pipefail

DATABASE_URL="${DATABASE_URL:-}"
if [[ $# -ge 1 ]]; then
  DATABASE_URL="$1"
fi

if [[ -z "${DATABASE_URL}" ]]; then
  echo "DATABASE_URL is required (either env var or first argument)." >&2
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MIGRATIONS_DIR="${SCRIPT_DIR}/../db/migrations"

docker run --rm \
  -v "${MIGRATIONS_DIR}:/migrations:ro" \
  migrate/migrate \
  -path /migrations \
  -database "${DATABASE_URL}" \
  up

