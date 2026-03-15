#!/usr/bin/env bash
# Generate Go code from .proto files.
# Requires: protoc, protoc-gen-go, protoc-gen-go-grpc
# Usage: run from repo root: ./scripts/proto-gen.sh

set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
PROTO_ROOT="$REPO_ROOT/proto"

cd "$REPO_ROOT"

# Upload state service (used by workers to update video/upload via gRPC)
if [ -f "$PROTO_ROOT/upload/upload.proto" ]; then
  (cd "$PROTO_ROOT/upload" && \
   protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    -I . upload.proto)
  echo "Generated proto/upload/*.pb.go"
fi
