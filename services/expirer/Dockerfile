FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
WORKDIR services/expirer
RUN go mod download
RUN CGO_ENABLED=0 go build -o /expirer ./cmd/server

FROM alpine:latest
RUN apk add --no-cache curl netcat-openbsd
COPY --from=builder /expirer /expirer
ENTRYPOINT ["/expirer"]

