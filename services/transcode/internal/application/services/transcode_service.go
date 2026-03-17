package service

import (
	"context"
	"fmt"
	"os"

	"github.com/LgAcerbi/go-video-upload/pkg/models"
	"github.com/LgAcerbi/go-video-upload/services/transcode/internal/application/ports"
)

const stepTranscode = "transcode"

var defaultLadder = []int{1080, 720, 480, 360}

// ComputeLadder returns resolution heights strictly less than sourceHeight (original is already stored).
func ComputeLadder(sourceHeight int) []int {
	var out []int
	for _, h := range defaultLadder {
		if h < sourceHeight {
			out = append(out, h)
		}
	}
	return out
}

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
	path, cleanup, err := s.fetcher.FetchToTempFile(ctx, s.bucket, storagePath)
	if err != nil {
		s.reportFailed(ctx, uploadID, videoID, storagePath, err)
		return err
	}
	defer cleanup()

	_, height, err := s.prober.Probe(ctx, path)
	if err != nil {
		s.reportFailed(ctx, uploadID, videoID, storagePath, err)
		return err
	}

	ladder := ComputeLadder(height)
	for _, h := range ladder {
		outputPath, cleanupOut, err := s.transcoder.Transcode(ctx, path, h)
		if err != nil {
			s.reportFailed(ctx, uploadID, videoID, storagePath, fmt.Errorf("transcode %dp: %w", h, err))
			return err
		}
		key := fmt.Sprintf("videos/%s/%dp.mp4", videoID, h)
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
