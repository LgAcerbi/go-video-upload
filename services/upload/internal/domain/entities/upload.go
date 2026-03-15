package entities

import (
	"time"

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
	UploadStatusPending    = "pending"
	UploadStatusProcessing = "processing"
	UploadStatusFinished   = "finished"
	UploadStatusFailed     = "failed"
	UploadStatusExpired    = "expired"
)

func NewUpload(videoID string, expiresAt *time.Time) *Upload {
	now := time.Now()
	return &Upload{
		ID:        uuid.New().String(),
		VideoID:   videoID,
		Status:    UploadStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: expiresAt,
	}
}
