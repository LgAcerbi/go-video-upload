package upload

// Request/response types for UploadStateService (workers call these via gRPC).
// Source of truth: upload.proto. Regenerate with: ./scripts/proto-gen.sh

type UpdateUploadStatusRequest struct {
	UploadId string
	Status   string
}

type UpdateUploadStatusResponse struct{}

type UpdateUploadStepRequest struct {
	UploadId     string
	Step         string
	Status       string
	ErrorMessage string
}

type UpdateUploadStepResponse struct{}

type UpdateVideoMetadataRequest struct {
	VideoId     string
	Format      string
	DurationSec float64
	Status      string
}

type UpdateVideoMetadataResponse struct{}

type CreateUploadStepsRequest struct {
	UploadId string
	Steps    []string
}

type CreateUploadStepsResponse struct{}
