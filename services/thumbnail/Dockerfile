FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
WORKDIR services/thumbnail
RUN go mod download
RUN CGO_ENABLED=0 go build -o /thumbnail ./cmd/server

FROM alpine:latest
RUN apk add --no-cache ffmpeg curl netcat-openbsd
COPY --from=builder /thumbnail /thumbnail
ENTRYPOINT ["/thumbnail"]

