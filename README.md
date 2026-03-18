# Go Video Upload Monorepo

Go monorepo for video-processing microservices.

## Quick start (run entire project)

### Prerequisites

- Docker Desktop (Docker Compose v2)

### Start

```bash
# From repo root
docker compose up -d --build
```

### First-time database schema

Run this once (or any time you recreate the Postgres volume):

```bash
docker compose --profile tools run --rm db-schema
```

### Stop

```bash
docker compose down
```

To also delete data volumes (Postgres/Influx/Grafana):

```bash
docker compose down -v
```

### Useful URLs (host)

- Upload API: `http://localhost:8080`
- Upload gRPC: `localhost:9090`
- E2E client UI: `http://localhost:3000`
- Adminer (Postgres UI): `http://localhost:8081`
- Grafana: `http://localhost:3001`
- InfluxDB: `http://localhost:8086`
- LocalStack (S3): `http://localhost:4566`
- RabbitMQ UI: `http://localhost:15672` (user/pass: `admin` / `admin`)

## Structure

- **services/upload** — REST API for uploads
- **services/metadata** — gRPC service for metadata extraction
- **services/transcode** — service for transcoding
- **pkg/** — Shared libraries (config, logger, middleware, models)
- **proto/** — Protobuf definitions for gRPC contracts

## Prerequisites

- Go 1.24+

## Commands

```bash
# Run tests
go test ./...

# Lint
golangci-lint run ./...

# Generate Go code from protos
./scripts/proto-gen.sh
```

## Development

From repo root, `go.work` enables local development: services and `pkg` are used as workspace modules. You can run each service from its directory with `go run ./cmd/server` or use Docker Compose as described above.
