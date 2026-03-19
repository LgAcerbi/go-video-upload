FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
WORKDIR services/outbox-dispatcher
RUN go mod download
RUN CGO_ENABLED=0 go build -o /outbox-dispatcher ./cmd/server

FROM alpine:latest
RUN apk add --no-cache curl netcat-openbsd
COPY --from=builder /outbox-dispatcher /outbox-dispatcher
ENTRYPOINT ["/outbox-dispatcher"]
