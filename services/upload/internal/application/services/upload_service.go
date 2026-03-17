package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/LgAcerbi/go-video-upload/services/upload/internal/application/ports"
	"github.com/LgAcerbi/go-video-upload/services/upload/internal/domain/entities"
)

var ErrInvalidPresignRequest = errors.New("invalid presign request")

var ErrInvalidExtension = errors.New("invalid file extension: only mp4 is allowed")

var allowedExtensions = []string{".mp4"}

func validateUploadExtension(filename string) error {
	if filename == "" {
		return errors.New("filename is required")
	}
	ext := strings.ToLower(filepath.Ext(filename))
	for _, allowed := range allowedExtensions {
		if ext == allowed {
			return nil
		}
	}
	return ErrInvalidExtension
}

const PresignExpiry = time.Hour

const originalObjectKeyPrefix = "videos/%s/original"

type UploadService struct {
	storage          ports.FileStorageRepository
	bucket           string
	videoRepo        ports.VideoRepository
	uploadRepo       ports.UploadRepository
	uploadStepRepo   ports.UploadStepRepository
	renditionRepo    ports.RenditionRepository
	uploadProcessPub ports.UploadProcessPublisher
}

func NewUploadService(storage ports.FileStorageRepository, bucket string, videoRepo ports.VideoRepository, uploadRepo ports.UploadRepository, uploadStepRepo ports.UploadStepRepository, renditionRepo ports.RenditionRepository, uploadProcessPub ports.UploadProcessPublisher) *UploadService {
	return &UploadService{
		storage:          storage,
		bucket:           bucket,
		videoRepo:        videoRepo,
		uploadRepo:       uploadRepo,
		uploadStepRepo:   uploadStepRepo,
		renditionRepo:    renditionRepo,
		uploadProcessPub: uploadProcessPub,
	}
}

func (s *UploadService) ValidateFile(filename string) error {
	return validateUploadExtension(filename)
}

func (s *UploadService) UploadFile(ctx context.Context, filename string, body io.Reader, contentLength int64, contentType string) (string, error) {
	if err := validateUploadExtension(filename); err != nil {
		return "", err
	}
	ext := filepath.Ext(filename)
	key := uuid.New().String() + ext
	input := &ports.UploadInput{
		Bucket:        s.bucket,
		Key:           key,
		Body:          body,
		ContentType:   contentType,
		ContentLength: contentLength,
	}
	if err := s.storage.Upload(ctx, input); err != nil {
		return "", err
	}
	return key, nil
}

func (s *UploadService) RequestPresignURL(ctx context.Context, userID, title string) (uploadURL, videoID string, err error) {
	if userID == "" {
		return "", "", fmt.Errorf("%w: user_id is required", ErrInvalidPresignRequest)
	}
	if _, err := uuid.Parse(userID); err != nil {
		return "", "", fmt.Errorf("%w: user_id must be a valid UUID", ErrInvalidPresignRequest)
	}
	video := entities.NewVideo(userID, title)
	if err := s.videoRepo.Create(ctx, video); err != nil {
		return "", "", fmt.Errorf("create video: %w", err)
	}
	expiresAt := time.Now().Add(PresignExpiry)
	upload := entities.NewUpload(video.ID, &expiresAt)
	if err := s.uploadRepo.Create(ctx, upload); err != nil {
		return "", "", fmt.Errorf("create upload: %w", err)
	}
	key := fmt.Sprintf(originalObjectKeyPrefix, video.ID)
	url, err := s.storage.PresignPut(ctx, s.bucket, key, PresignExpiry)
	if err != nil {
		return "", "", fmt.Errorf("presign put: %w", err)
	}
	return url, video.ID, nil
}

var ErrFinalizeUpload = errors.New("cannot finalize upload")

func (s *UploadService) FinalizeUpload(ctx context.Context, videoID string) error {
	if videoID == "" {
		return fmt.Errorf("%w: video_id is required", ErrFinalizeUpload)
	}
	upload, err := s.uploadRepo.GetByVideoID(ctx, videoID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrFinalizeUpload, err)
	}
	if upload.Status != entities.UploadStatusPending {
		return fmt.Errorf("%w: upload is not pending (status=%s)", ErrFinalizeUpload, upload.Status)
	}
	storagePath := fmt.Sprintf(originalObjectKeyPrefix, videoID)
	upload.StoragePath = storagePath
	upload.Status = entities.UploadStatusProcessing
	upload.UpdatedAt = time.Now()
	if err := s.uploadRepo.Update(ctx, upload); err != nil {
		return fmt.Errorf("update upload: %w", err)
	}
	if err := s.uploadProcessPub.PublishUploadProcess(ctx, videoID, upload.ID, storagePath); err != nil {
		return fmt.Errorf("publish to upload-process queue: %w", err)
	}
	return nil
}

func (s *UploadService) UpdateUploadStatus(ctx context.Context, uploadID, status string) error {
	if uploadID == "" || status == "" {
		return nil
	}
	return s.uploadRepo.UpdateStatus(ctx, uploadID, status)
}

func (s *UploadService) UpdateUploadStep(ctx context.Context, uploadID, step, status, errorMessage string) error {
	if uploadID == "" || step == "" || status == "" {
		return nil
	}
	return s.uploadStepRepo.UpdateStepStatus(ctx, uploadID, step, status, errorMessage)
}

func (s *UploadService) UpdateVideoMetadata(ctx context.Context, videoID, format string, durationSec float64, status string) error {
	if videoID == "" {
		return nil
	}
	v, err := s.videoRepo.GetByID(ctx, videoID)
	if err != nil {
		return err
	}
	if format != "" {
		v.Format = format
	}
	if durationSec > 0 {
		v.DurationSec = &durationSec
	}
	if status != "" {
		v.Status = status
	}
	v.UpdatedAt = time.Now()
	return s.videoRepo.Update(ctx, v)
}

func (s *UploadService) CreateUploadSteps(ctx context.Context, uploadID string, steps []string) error {
	if uploadID == "" || len(steps) == 0 {
		return nil
	}
	return s.uploadStepRepo.CreateSteps(ctx, uploadID, steps)
}

func (s *UploadService) ListUploads(ctx context.Context, limit int) ([]*entities.Upload, error) {
	return s.uploadRepo.ListAll(ctx, limit)
}

func (s *UploadService) ListVideos(ctx context.Context, limit int) ([]*entities.Video, error) {
	return s.videoRepo.ListAll(ctx, limit)
}

func (s *UploadService) CreateRenditions(ctx context.Context, videoID, originalStoragePath string, originalWidth, originalHeight int32, targetHeights []int32) error {
	if videoID == "" {
		return nil
	}
	heights := make([]int, len(targetHeights))
	for i, h := range targetHeights {
		heights[i] = int(h)
	}
	return s.renditionRepo.CreateBatch(ctx, videoID, originalStoragePath, int(originalWidth), int(originalHeight), heights)
}

func (s *UploadService) ListPendingRenditions(ctx context.Context, videoID string) ([]*entities.Rendition, error) {
	if videoID == "" {
		return nil, nil
	}
	return s.renditionRepo.ListPendingByVideoID(ctx, videoID)
}

func (s *UploadService) UpdateRendition(ctx context.Context, videoID, resolution, storagePath string, width, height, bitrateKbps *int32) error {
	if videoID == "" || resolution == "" || storagePath == "" {
		return nil
	}
	r := &entities.Rendition{
		VideoID:     videoID,
		Resolution:  resolution,
		StoragePath: &storagePath,
		Status:      entities.RenditionStatusReady,
	}
	if width != nil && *width > 0 {
		w := int(*width)
		r.Width = &w
	}
	if height != nil && *height > 0 {
		h := int(*height)
		r.Height = &h
	}
	if bitrateKbps != nil && *bitrateKbps > 0 {
		b := int(*bitrateKbps)
		r.BitrateKbps = &b
	}
	return s.renditionRepo.Update(ctx, r)
}
