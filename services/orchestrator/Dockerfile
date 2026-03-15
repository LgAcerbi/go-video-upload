FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
WORKDIR services/orchestrator
RUN go mod download
RUN CGO_ENABLED=0 go build -o /orchestrator ./cmd/server

FROM alpine:latest
COPY --from=builder /orchestrator /orchestrator
ENTRYPOINT ["/orchestrator"]
