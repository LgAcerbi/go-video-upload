package service

import (
	"context"
	"fmt"
	"os"

	"github.com/LgAcerbi/go-video-upload/pkg/models"
	"github.com/LgAcerbi/go-video-upload/services/thumbnail/internal/application/ports"
)

const stepGenerateThumbnail = "generate_thumbnail"

type ThumbnailService struct {
	uploadClient ports.UploadStateClient
	fetcher      ports.VideoFileFetcher
	uploader     ports.ThumbnailUploader
	generator    ports.ThumbnailGenerator
	stepPub      ports.StepResultPublisher
	bucket       string
}

func NewThumbnailService(
	uploadClient ports.UploadStateClient,
	fetcher ports.VideoFileFetcher,
	uploader ports.ThumbnailUploader,
	generator ports.ThumbnailGenerator,
	stepPub ports.StepResultPublisher,
	bucket string,
) *ThumbnailService {
	return &ThumbnailService{
		uploadClient: uploadClient,
		fetcher:      fetcher,
		uploader:     uploader,
		generator:    generator,
		stepPub:      stepPub,
		bucket:       bucket,
	}
}

func (s *ThumbnailService) GenerateThumbnail(ctx context.Context, uploadID string) error {
	ctxData, err := s.uploadClient.GetUploadProcessingContext(ctx, uploadID)
	if err != nil {
		return err
	}
	videoID := ctxData.VideoID
	storagePath := ctxData.StoragePath

	inputPath, cleanupIn, err := s.fetcher.FetchToTempFile(ctx, s.bucket, storagePath)
	if err != nil {
		return err
	}
	defer cleanupIn()

	outPath, cleanupOut, err := s.generator.Generate(ctx, inputPath)
	if err != nil {
		return err
	}
	defer cleanupOut()

	key := fmt.Sprintf("videos/%s/thumbnail.jpg", videoID)
	fi, err := os.Stat(outPath)
	if err != nil {
		return err
	}
	f, err := os.Open(outPath)
	if err != nil {
		return err
	}
	if err := s.uploader.UploadThumbnail(ctx, s.bucket, key, f, fi.Size()); err != nil {
		f.Close()
		return err
	}
	f.Close()

	if err := s.uploadClient.UpdateVideoThumbnail(ctx, videoID, key); err != nil {
		return err
	}
	stepRes, err := s.uploadClient.UpdateUploadStep(ctx, uploadID, stepGenerateThumbnail, "done", "")
	if err != nil {
		return err
	}
	if !stepRes.Applied {
		return nil
	}
	if err := s.stepPub.PublishStepResult(ctx, uploadID, stepGenerateThumbnail, "done", ""); err != nil {
		return err
	}
	return nil
}

func (s *ThumbnailService) ReportFailed(ctx context.Context, uploadID string, err error) {
	errMsg := err.Error()
	stepRes, updateErr := s.uploadClient.UpdateUploadStep(ctx, uploadID, stepGenerateThumbnail, models.UploadStatusFailed, errMsg)
	if updateErr != nil || !stepRes.Applied {
		return
	}
	_ = s.stepPub.PublishStepResult(ctx, uploadID, stepGenerateThumbnail, models.UploadStatusFailed, errMsg)
}
