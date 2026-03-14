.PHONY: build-all test-all lint proto-gen build-upload build-metadata build-transcoding

build-all: build-upload build-metadata build-transcoding

build-upload:
	go build -o bin/upload ./services/upload/cmd/server

build-metadata:
	go build -o bin/metadata ./services/metadata/cmd/server

build-transcoding:
	go build -o bin/transcoding ./services/transcoding/cmd/server

test-all:
	go test ./...

lint:
	golangci-lint run ./...

proto-gen:
	./scripts/proto-gen.sh
