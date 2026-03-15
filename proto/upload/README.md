# proto/upload

gRPC API for the upload state service. Workers and the orchestrator call this to update video/upload/upload_steps (only the upload service writes to the DB).

- **types.go** – request/response structs (hand-written so the project builds without `protoc`).
- **upload_grpc.pb.go** – server interface and registration.
- **upload.proto** – source of truth; run `./scripts/proto-gen.sh` (requires `protoc`, `protoc-gen-go`, `protoc-gen-go-grpc`) to generate canonical `upload.pb.go` for proper protobuf wire format. Until then, the server runs with the hand-written types.

Workers can depend on this module and use `upload.NewUploadStateServiceClient(cc)` to get a client.
