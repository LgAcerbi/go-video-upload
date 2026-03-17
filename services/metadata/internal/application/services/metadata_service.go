package service

import (
	"context"

	"github.com/LgAcerbi/go-video-upload/pkg/models"
	"github.com/LgAcerbi/go-video-upload/services/metadata/internal/application/ports"
)

const stepExtractMetadata = "extract_metadata"

type MetadataService struct {
	uploadClient   ports.UploadStateClient
	stepResultPub  ports.StepResultPublisher
	fileFetcher    ports.VideoFileFetcher
	metadataExtractor ports.MetadataExtractor
	bucket         string
}

func NewMetadataService(
	uploadClient ports.UploadStateClient,
	stepResultPub ports.StepResultPublisher,
	fileFetcher ports.VideoFileFetcher,
	metadataExtractor ports.MetadataExtractor,
	bucket string,
) *MetadataService {
	return &MetadataService{
		uploadClient:       uploadClient,
		stepResultPub:      stepResultPub,
		fileFetcher:        fileFetcher,
		metadataExtractor:  metadataExtractor,
		bucket:             bucket,
	}
}

func (s *MetadataService) ExtractMetadata(ctx context.Context, videoID, uploadID, storagePath string) error {
	path, cleanup, err := s.fileFetcher.FetchToTempFile(ctx, s.bucket, storagePath)
	if err != nil {
		s.reportFailed(ctx, uploadID, videoID, storagePath, err)
		return err
	}
	defer cleanup()

	format, durationSec, width, height, err := s.metadataExtractor.Extract(ctx, path)
	if err != nil {
		s.reportFailed(ctx, uploadID, videoID, storagePath, err)
		return err
	}

	if err := s.uploadClient.UpdateVideoMetadata(ctx, videoID, format, durationSec, ""); err != nil {
		s.reportFailed(ctx, uploadID, videoID, storagePath, err)
		return err
	}
	ladder := computeLadder(int(height))
	if err := s.uploadClient.CreateRenditions(ctx, videoID, storagePath, width, height, ladder); err != nil {
		s.reportFailed(ctx, uploadID, videoID, storagePath, err)
		return err
	}
	if err := s.uploadClient.UpdateUploadStep(ctx, uploadID, stepExtractMetadata, "done", ""); err != nil {
		s.reportFailed(ctx, uploadID, videoID, storagePath, err)
		return err
	}
	return s.stepResultPub.PublishStepResult(ctx, uploadID, videoID, stepExtractMetadata, "done", "", storagePath)
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

func (s *MetadataService) reportFailed(ctx context.Context, uploadID, videoID, storagePath string, err error) {
	errMsg := err.Error()
	_ = s.uploadClient.UpdateUploadStep(ctx, uploadID, stepExtractMetadata, models.UploadStatusFailed, errMsg)
	_ = s.stepResultPub.PublishStepResult(ctx, uploadID, videoID, stepExtractMetadata, models.UploadStatusFailed, errMsg, storagePath)
}
