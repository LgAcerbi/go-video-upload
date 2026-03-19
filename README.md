# Go Video Upload Monorepo

This repo is a **study case in Go** for building a small **video upload → process → publish** system. It stitches together several **microservices architecture** ideas so you can see them work end to end in one codebase—not production polish, but a hands-on map of common patterns.

**Approaches and patterns included here:**

- **Microservices / bounded contexts** — upload API, orchestrator, workers (metadata, thumbnail, transcode, segment, publish), and outbox dispatcher as separate services
- **gRPC + Protocol Buffers** — contracts under `proto/`, including the upload state API consumed by workers
- **REST HTTP** — client-facing upload API (e.g. presign and finalize)
- **Transactional outbox** — `outbox_events` written in the same transaction as domain changes, then relayed to the broker by `outbox-dispatcher`
- **Message-driven pipeline** — RabbitMQ queues/exchanges for `upload-process`, per-step work, and step completion feedback
- **Orchestration** — orchestrator service owns pipeline steps and progression across workers
- **Hexagonal & Clean Architecture patterns** — application ports with Postgres, AMQP, and object-storage adapters
- **Direct-to-object-storage uploads** — presigned URLs to S3-compatible storage so bytes bypass the API

## Quick start (run entire project)

### Prerequisites

- Docker Desktop (Docker Compose v2)

### Non-production defaults

This repository ships with development-friendly defaults in `docker-compose.yml` (for example admin credentials and dev tokens). Before using it on shared machines, VPNs, or any public network:

- copy `.env.example` to `.env` and set strong secrets
- do not expose infrastructure ports unless you need host access
- rotate credentials/tokens from local defaults

### Start

```bash
# From repo root
docker compose up -d --build
```

Optional setup for custom secrets:

```bash
cp .env.example .env
```

Optional hardened local profile (example):

```bash
# after creating docker-compose.override.yml from docker-compose.override.example.yml
docker compose --profile internal-infra up -d --build
```

### Database schema / migrations

DB migrations run automatically on every `docker compose up` (via the `db-migrate` one-shot service).

If you want to run migrations manually (for debugging), you can also use:

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

### E2E client API key note

The e2e frontend now sends `X-Api-Key` using `VITE_UPLOAD_API_KEY`. In Compose, both upload `SECRET_TOKEN` and e2e `VITE_UPLOAD_API_KEY` are sourced from `API_SHARED_TOKEN` so they stay aligned. This is intended for local e2e/dev only; embedding API keys in SPA assets is not a production authentication model.

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