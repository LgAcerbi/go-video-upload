package service

import (
	"context"
	"fmt"

	"github.com/LgAcerbi/go-video-upload/pkg/models"
	"github.com/LgAcerbi/go-video-upload/services/segment/internal/application/ports"
)

const stepSegment = "segment"

type SegmentService struct {
	uploadClient ports.UploadStateClient
	fetcher      ports.VideoFileFetcher
	producer     ports.HlsProducer
	stepPub      ports.StepResultPublisher
	bucket       string
}

func NewSegmentService(
	uploadClient ports.UploadStateClient,
	fetcher ports.VideoFileFetcher,
	producer ports.HlsProducer,
	stepPub ports.StepResultPublisher,
	bucket string,
) *SegmentService {
	return &SegmentService{
		uploadClient: uploadClient,
		fetcher:      fetcher,
		producer:     producer,
		stepPub:      stepPub,
		bucket:       bucket,
	}
}

func (s *SegmentService) Segment(ctx context.Context, uploadID string) error {
	ctxData, err := s.uploadClient.GetUploadProcessingContext(ctx, uploadID)
	if err != nil {
		s.reportFailed(ctx, uploadID, err)
		return err
	}
	videoID := ctxData.VideoID

	ready, err := s.uploadClient.ListReadyRenditions(ctx, videoID)
	if err != nil {
		s.reportFailed(ctx, uploadID, err)
		return err
	}
	if len(ready) == 0 {
		if err := s.uploadClient.UpdateUploadStep(ctx, uploadID, stepSegment, "done", ""); err != nil {
			s.reportFailed(ctx, uploadID, err)
			return err
		}
		return s.stepPub.PublishStepResult(ctx, uploadID, stepSegment, "done", "")
	}

	for _, rend := range ready {
		if rend.StoragePath == "" {
			continue
		}
		path, cleanup, err := s.fetcher.FetchToTempFile(ctx, s.bucket, rend.StoragePath)
		if err != nil {
			s.reportFailed(ctx, uploadID, fmt.Errorf("fetch %s: %w", rend.Resolution, err))
			return err
		}
		outputPrefix := "videos/" + videoID + "/hls/" + rend.Resolution
		if err := s.producer.ProduceAndUpload(ctx, s.bucket, outputPrefix, path); err != nil {
			cleanup()
			s.reportFailed(ctx, uploadID, fmt.Errorf("hls %s: %w", rend.Resolution, err))
			return err
		}
		cleanup()
	}

	if err := s.uploadClient.UpdateUploadStep(ctx, uploadID, stepSegment, "done", ""); err != nil {
		s.reportFailed(ctx, uploadID, err)
		return err
	}
	return s.stepPub.PublishStepResult(ctx, uploadID, stepSegment, "done", "")
}

func (s *SegmentService) reportFailed(ctx context.Context, uploadID string, err error) {
	errMsg := err.Error()
	_ = s.uploadClient.UpdateUploadStep(ctx, uploadID, stepSegment, models.UploadStatusFailed, errMsg)
	_ = s.stepPub.PublishStepResult(ctx, uploadID, stepSegment, models.UploadStatusFailed, errMsg)
}
