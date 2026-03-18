# Regenerate proto Go files using Docker (same as make proto-gen-docker).
# Run from repo root: .\scripts\proto-gen-docker.ps1
$ErrorActionPreference = "Stop"
$root = (Get-Location).Path
docker run --rm -v "${root}:/app" -w /app golang:1.22-bookworm sh -c "apt-get update -qq && apt-get install -y -qq protobuf-compiler 2>/dev/null && go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.31.0 && go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0 && cd proto/upload && protoc -I . --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative upload.proto"
Write-Host "Generated proto/upload/*.pb.go"
