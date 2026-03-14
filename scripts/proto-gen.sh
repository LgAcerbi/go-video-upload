#!/usr/bin/env bash
# Generate Go code from .proto files.
# Requires: protoc, protoc-gen-go, protoc-gen-go-grpc
# Usage: run from repo root: ./scripts/proto-gen.sh

set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
PROTO_DIR="$REPO_ROOT/proto"

cd "$REPO_ROOT"
# Placeholder: add protoc invocations per proto package when ready
# Example: protoc --go_out=. --go-grpc_out=. -I "$PROTO_DIR" "$PROTO_DIR/metadata/*.proto"
echo "Proto generation placeholder. Configure protoc commands for metadata and transcoding protos."
