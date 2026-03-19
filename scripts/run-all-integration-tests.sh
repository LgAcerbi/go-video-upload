#!/usr/bin/env bash
set -euo pipefail

GO_TEST_TAGS="${1:-integration}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

paths=(
  "./services/upload/test/integration"
  "./services/metadata/test/integration"
  "./services/orchestrator/test/integration"
  "./services/transcode/test/integration"
  "./services/segment/test/integration"
  "./services/thumbnail/test/integration"
  "./services/publish/test/integration"
  "./services/expirer/cmd/server"
  "./services/outbox-dispatcher/cmd/server"
)

for p in "${paths[@]}"; do
  echo "==> go test -tags=${GO_TEST_TAGS} -v -count=1 ${p}"
  go test -tags="${GO_TEST_TAGS}" -v -count=1 "${p}"
done

