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
	storage      ports.FileStorageRepository
	bucket       string
	videoRepo    ports.VideoRepository
	uploadRepo   ports.UploadRepository
}

func NewUploadService(storage ports.FileStorageRepository, bucket string, videoRepo ports.VideoRepository, uploadRepo ports.UploadRepository) *UploadService {
	return &UploadService{
		storage:    storage,
		bucket:     bucket,
		videoRepo:  videoRepo,
		uploadRepo: uploadRepo,
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
