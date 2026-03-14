# Go Video Upload Monorepo

Go monorepo for video-processing microservices.

## Structure

- **services/upload** — REST API for uploads
- **services/metadata** — gRPC service for metadata extraction
- **services/transcoding** — gRPC service for transcoding
- **pkg/** — Shared libraries (config, logger, middleware, models)
- **proto/** — Protobuf definitions for gRPC contracts

## Prerequisites

- Go 1.24+
- Make (optional, for Makefile targets)

## Commands

```bash
# Build all services
make build-all

# Build individual service
make build-upload
make build-metadata
make build-transcoding

# Run tests
make test-all

# Lint
make lint

# Generate Go code from protos
make proto-gen
```

## Development

From repo root, `go.work` enables local development: services and `pkg` are used as workspace modules. Run each service from its directory or via the Makefile.
