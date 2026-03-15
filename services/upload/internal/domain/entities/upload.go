package entities

import (
	"time"

	"github.com/LgAcerbi/go-video-upload/pkg/models"
	"github.com/google/uuid"
)

type Upload struct {
	ID          string
	VideoID     string
	StoragePath string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
	ExpiresAt   *time.Time
}

const (
	UploadStatusPending    = models.UploadStatusPending
	UploadStatusProcessing = models.UploadStatusProcessing
	UploadStatusFinished   = models.UploadStatusFinished
	UploadStatusFailed     = models.UploadStatusFailed
	UploadStatusExpired    = models.UploadStatusExpired
)

func NewUpload(videoID string, expiresAt *time.Time) *Upload {
	now := time.Now()
	return &Upload{
		ID:        uuid.New().String(),
		VideoID:   videoID,
		Status:    models.UploadStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: expiresAt,
	}
}
