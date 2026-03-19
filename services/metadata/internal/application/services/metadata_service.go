package service

import (
	"context"

	"github.com/LgAcerbi/go-video-upload/pkg/models"
	"github.com/LgAcerbi/go-video-upload/services/metadata/internal/application/ports"
)

const stepExtractMetadata = "extract_metadata"

type MetadataService struct {
	uploadClient      ports.UploadStateClient
	stepResultPub     ports.StepResultPublisher
	fileFetcher       ports.VideoFileFetcher
	metadataExtractor ports.MetadataExtractor
	bucket            string
}

func NewMetadataService(
	uploadClient ports.UploadStateClient,
	stepResultPub ports.StepResultPublisher,
	fileFetcher ports.VideoFileFetcher,
	metadataExtractor ports.MetadataExtractor,
	bucket string,
) *MetadataService {
	return &MetadataService{
		uploadClient:      uploadClient,
		stepResultPub:     stepResultPub,
		fileFetcher:       fileFetcher,
		metadataExtractor: metadataExtractor,
		bucket:            bucket,
	}
}

func (s *MetadataService) ExtractMetadata(ctx context.Context, uploadID string) error {
	ctxData, err := s.uploadClient.GetUploadProcessingContext(ctx, uploadID)
	if err != nil {
		return err
	}
	videoID := ctxData.VideoID
	storagePath := ctxData.StoragePath

	path, cleanup, err := s.fileFetcher.FetchToTempFile(ctx, s.bucket, storagePath)
	if err != nil {
		return err
	}
	defer cleanup()

	format, durationSec, width, height, err := s.metadataExtractor.Extract(ctx, path)
	if err != nil {
		return err
	}

	if err := s.uploadClient.UpdateVideoMetadata(ctx, videoID, format, durationSec, ""); err != nil {
		return err
	}
	ladder := computeLadder(int(height))
	if err := s.uploadClient.CreateRenditions(ctx, videoID, storagePath, width, height, ladder); err != nil {
		return err
	}
	stepRes, err := s.uploadClient.UpdateUploadStep(ctx, uploadID, stepExtractMetadata, "done", "")
	if err != nil {
		return err
	}
	if !stepRes.Applied {
		return nil
	}
	return s.stepResultPub.PublishStepResult(ctx, uploadID, stepExtractMetadata, "done", "")
}

var defaultLadder = []int32{1080, 720, 480, 360}

func computeLadder(sourceHeight int) []int32 {
	var out []int32
	for _, h := range defaultLadder {
		if h < int32(sourceHeight) {
			out = append(out, h)
		}
	}
	return out
}

func (s *MetadataService) ReportFailed(ctx context.Context, uploadID string, err error) {
	errMsg := err.Error()
	stepRes, updateErr := s.uploadClient.UpdateUploadStep(ctx, uploadID, stepExtractMetadata, models.UploadStatusFailed, errMsg)
	if updateErr != nil || !stepRes.Applied {
		return
	}
	_ = s.stepResultPub.PublishStepResult(ctx, uploadID, stepExtractMetadata, models.UploadStatusFailed, errMsg)
}
