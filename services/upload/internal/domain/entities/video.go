package entities

import (
	"time"

	"github.com/google/uuid"
)

type Video struct {
	ID          string
	UserID      string
	Title       string
	Format      string
	ThumbnailPath string
	Status      string
	DurationSec *float64
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

const (
	VideoStatusProcessing = "processing"
	VideoStatusReady      = "ready"
	VideoStatusFailed     = "failed"
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
