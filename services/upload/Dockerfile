FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
WORKDIR services/upload
RUN go mod download
RUN CGO_ENABLED=0 go build -o /upload ./cmd/server

FROM alpine:latest
COPY --from=builder /upload /upload
EXPOSE 8080
ENTRYPOINT ["/upload"]
