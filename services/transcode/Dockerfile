FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
WORKDIR services/transcode
RUN go mod download
RUN CGO_ENABLED=0 go build -o /transcode ./cmd/server

FROM alpine:latest
RUN apk add --no-cache ffmpeg
COPY --from=builder /transcode /transcode
ENTRYPOINT ["/transcode"]
