package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/LgAcerbi/go-video-upload/services/upload/internal/domain"
	"github.com/LgAcerbi/go-video-upload/services/upload/internal/ports"
)

var ErrInvalidPresignRequest = errors.New("invalid presign request")

const PresignExpiry = time.Hour

const originalObjectKeyPrefix = "videos/%s/original"

type UploadService struct {
	storage           ports.FileStorageRepository
	bucket            string
	videoRepo         ports.VideoRepository
	uploadRepo        ports.UploadRepository
	uploadProcessPub  ports.UploadProcessPublisher
}

func NewUploadService(storage ports.FileStorageRepository, bucket string, videoRepo ports.VideoRepository, uploadRepo ports.UploadRepository, uploadProcessPub ports.UploadProcessPublisher) *UploadService {
	return &UploadService{
		storage:          storage,
		bucket:           bucket,
		videoRepo:        videoRepo,
		uploadRepo:       uploadRepo,
		uploadProcessPub: uploadProcessPub,
	}
}

func (s *UploadService) ValidateFile(filename string) error {
	return domain.ValidateUploadExtension(filename)
}

func (s *UploadService) UploadFile(ctx context.Context, filename string, body io.Reader, contentLength int64, contentType string) (string, error) {
	if err := domain.ValidateUploadExtension(filename); err != nil {
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
	video := domain.NewVideo(userID, title)
	if err := s.videoRepo.Create(ctx, video); err != nil {
		return "", "", fmt.Errorf("create video: %w", err)
	}
	expiresAt := time.Now().Add(PresignExpiry)
	upload := domain.NewUpload(video.ID, &expiresAt)
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
	if upload.Status != domain.UploadStatusPending {
		return fmt.Errorf("%w: upload is not pending (status=%s)", ErrFinalizeUpload, upload.Status)
	}
	storagePath := fmt.Sprintf(originalObjectKeyPrefix, videoID)
	upload.StoragePath = storagePath
	upload.Status = domain.UploadStatusProcessing
	upload.UpdatedAt = time.Now()
	if err := s.uploadRepo.Update(ctx, upload); err != nil {
		return fmt.Errorf("update upload: %w", err)
	}
	if err := s.uploadProcessPub.PublishUploadProcess(ctx, videoID, upload.ID, storagePath); err != nil {
		return fmt.Errorf("publish to upload-process queue: %w", err)
	}
	return nil
}
