FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
WORKDIR services/metadata
RUN go mod download
RUN CGO_ENABLED=0 go build -o /metadata ./cmd/server

FROM alpine:latest
RUN apk add --no-cache ffmpeg
COPY --from=builder /metadata /metadata
EXPOSE 50051
ENTRYPOINT ["/metadata"]
