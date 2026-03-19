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

func (s *TranscodeService) Transcode(ctx context.Context, uploadID string) error {
	ctxData, err := s.uploadClient.GetUploadProcessingContext(ctx, uploadID)
	if err != nil {
		return err
	}
	videoID := ctxData.VideoID
	storagePath := ctxData.StoragePath

	pending, err := s.uploadClient.ListPendingRenditions(ctx, videoID)
	if err != nil {
		return err
	}
	if len(pending) == 0 {
		stepRes, err := s.uploadClient.UpdateUploadStep(ctx, uploadID, stepTranscode, "done", "")
		if err != nil {
			return err
		}
		if !stepRes.Applied {
			return nil
		}
		return s.stepPub.PublishStepResult(ctx, uploadID, stepTranscode, "done", "")
	}

	path, cleanup, err := s.fetcher.FetchToTempFile(ctx, s.bucket, storagePath)
	if err != nil {
		return err
	}
	defer cleanup()

	for _, rend := range pending {
		outputPath, cleanupOut, err := s.transcoder.Transcode(ctx, path, rend.Height)
		if err != nil {
			return err
		}
		key := fmt.Sprintf("videos/%s/%s.mp4", videoID, rend.Resolution)
		fi, err := os.Stat(outputPath)
		if err != nil {
			cleanupOut()
			return err
		}
		f, err := os.Open(outputPath)
		if err != nil {
			cleanupOut()
			return err
		}
		if err := s.uploader.UploadRendition(ctx, s.bucket, key, f, fi.Size()); err != nil {
			f.Close()
			cleanupOut()
			return err
		}
		f.Close()
		probedW, probedH, err := s.prober.Probe(ctx, outputPath)
		if err != nil {
			cleanupOut()
			return err
		}
		w32, h32 := int32(probedW), int32(probedH)
		cleanupOut()
		if err := s.uploadClient.UpdateRendition(ctx, videoID, rend.Resolution, key, &w32, &h32, nil, "mp4"); err != nil {
			return err
		}
	}

	stepRes, err := s.uploadClient.UpdateUploadStep(ctx, uploadID, stepTranscode, "done", "")
	if err != nil {
		return err
	}
	if !stepRes.Applied {
		return nil
	}
	if err := s.stepPub.PublishStepResult(ctx, uploadID, stepTranscode, "done", ""); err != nil {
		return err
	}
	return nil
}

func (s *TranscodeService) ReportFailed(ctx context.Context, uploadID string, err error) {
	errMsg := err.Error()
	stepRes, updateErr := s.uploadClient.UpdateUploadStep(ctx, uploadID, stepTranscode, models.UploadStatusFailed, errMsg)
	if updateErr != nil || !stepRes.Applied {
		return
	}
	_ = s.stepPub.PublishStepResult(ctx, uploadID, stepTranscode, models.UploadStatusFailed, errMsg)
}
