package domain

import (
	"time"

	"github.com/google/uuid"
)

type Video struct {
	ID          string
	UserID      string
	Title       string
	Format      string
	Status      string
	DurationSec *float64
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

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
	VideoStatusProcessing = "processing"
	VideoStatusReady      = "ready"
	VideoStatusFailed     = "failed"

	UploadStatusPending   = "pending"
	UploadStatusFinished  = "finished"
	UploadStatusFailed    = "failed"
	UploadStatusExpired   = "expired"
)

func NewVideo(userID, title string) *Video {
	now := time.Now()
	return &Video{
		ID:        uuid.New().String(),
		UserID:    userID,
		Title:     title,
		Status:    VideoStatusProcessing,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

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
