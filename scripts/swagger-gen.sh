#!/usr/bin/env bash
# Regenerate Swagger docs for the upload service.
# Run from repo root: ./scripts/swagger-gen.sh
# Requires: go install github.com/swaggo/swag/cmd/swag@latest

set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
UPLOAD_DIR="$REPO_ROOT/services/upload"

cd "$UPLOAD_DIR"
go run github.com/swaggo/swag/cmd/swag@v1.16.3 init -g cmd/server/main.go -d . -o docs --parseInternal
echo "Swagger docs generated in services/upload/docs/"
