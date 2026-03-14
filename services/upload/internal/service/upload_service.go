package service

import (
	"context"
	"io"
	"path/filepath"

	"github.com/google/uuid"

	"github.com/LgAcerbi/go-video-upload/services/upload/internal/domain"
	"github.com/LgAcerbi/go-video-upload/services/upload/internal/ports"
)

type UploadService struct {
	storage ports.FileStorage
	bucket  string
}

func NewUploadService(storage ports.FileStorage, bucket string) *UploadService {
	return &UploadService{storage: storage, bucket: bucket}
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
