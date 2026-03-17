package service

import (
	"context"
	"fmt"
	"os"

	"github.com/LgAcerbi/go-video-upload/pkg/models"
	"github.com/LgAcerbi/go-video-upload/services/transcode/internal/application/ports"
)

const stepTranscode = "transcode"

type TranscodeService struct {
	uploadClient ports.UploadStateClient
	fetcher      ports.VideoFileFetcher
	uploader     ports.RenditionUploader
	transcoder   ports.Transcoder
	prober       ports.DimensionsProber
	stepPub      ports.StepResultPublisher
	bucket       string
}

func NewTranscodeService(
	uploadClient ports.UploadStateClient,
	fetcher ports.VideoFileFetcher,
	uploader ports.RenditionUploader,
	transcoder ports.Transcoder,
	prober ports.DimensionsProber,
	stepPub ports.StepResultPublisher,
	bucket string,
) *TranscodeService {
	return &TranscodeService{
		uploadClient: uploadClient,
		fetcher:      fetcher,
		uploader:     uploader,
		transcoder:   transcoder,
		prober:       prober,
		stepPub:      stepPub,
		bucket:       bucket,
	}
}

func (s *TranscodeService) Transcode(ctx context.Context, videoID, uploadID, storagePath string) error {
	pending, err := s.uploadClient.ListPendingRenditions(ctx, videoID)
	if err != nil {
		s.reportFailed(ctx, uploadID, videoID, storagePath, err)
		return err
	}
	if len(pending) == 0 {
		if err := s.uploadClient.UpdateUploadStep(ctx, uploadID, stepTranscode, "done", ""); err != nil {
			s.reportFailed(ctx, uploadID, videoID, storagePath, err)
			return err
		}
		return s.stepPub.PublishStepResult(ctx, uploadID, videoID, stepTranscode, "done", "", storagePath)
	}

	path, cleanup, err := s.fetcher.FetchToTempFile(ctx, s.bucket, storagePath)
	if err != nil {
		s.reportFailed(ctx, uploadID, videoID, storagePath, err)
		return err
	}
	defer cleanup()

	for _, rend := range pending {
		outputPath, cleanupOut, err := s.transcoder.Transcode(ctx, path, rend.Height)
		if err != nil {
			s.reportFailed(ctx, uploadID, videoID, storagePath, fmt.Errorf("transcode %s: %w", rend.Resolution, err))
			return err
		}
		key := fmt.Sprintf("videos/%s/%s.mp4", videoID, rend.Resolution)
		fi, err := os.Stat(outputPath)
		if err != nil {
			cleanupOut()
			s.reportFailed(ctx, uploadID, videoID, storagePath, err)
			return err
		}
		f, err := os.Open(outputPath)
		if err != nil {
			cleanupOut()
			s.reportFailed(ctx, uploadID, videoID, storagePath, err)
			return err
		}
		if err := s.uploader.UploadRendition(ctx, s.bucket, key, f, fi.Size()); err != nil {
			f.Close()
			cleanupOut()
			s.reportFailed(ctx, uploadID, videoID, storagePath, err)
			return err
		}
		f.Close()
		cleanupOut()
		if err := s.uploadClient.UpdateRendition(ctx, videoID, rend.Resolution, key, nil, nil, nil); err != nil {
			s.reportFailed(ctx, uploadID, videoID, storagePath, err)
			return err
		}
	}

	if err := s.uploadClient.UpdateUploadStep(ctx, uploadID, stepTranscode, "done", ""); err != nil {
		s.reportFailed(ctx, uploadID, videoID, storagePath, err)
		return err
	}
	if err := s.stepPub.PublishStepResult(ctx, uploadID, videoID, stepTranscode, "done", "", storagePath); err != nil {
		s.reportFailed(ctx, uploadID, videoID, storagePath, err)
		return err
	}
	return nil
}

func (s *TranscodeService) reportFailed(ctx context.Context, uploadID, videoID, storagePath string, err error) {
	errMsg := err.Error()
	_ = s.uploadClient.UpdateUploadStep(ctx, uploadID, stepTranscode, models.UploadStatusFailed, errMsg)
	_ = s.stepPub.PublishStepResult(ctx, uploadID, videoID, stepTranscode, models.UploadStatusFailed, errMsg, storagePath)
}
