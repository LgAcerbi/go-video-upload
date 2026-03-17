package entities

import "time"

const (
	RenditionStatusPending = "pending"
	RenditionStatusReady   = "ready"
)

const ResolutionOriginal = "original"

type Rendition struct {
	ID           string
	VideoID      string
	Resolution   string
	StoragePath  *string
	Format       string
	Width        *int
	Height       *int
	BitrateKbps  *int
	Status       string
	CreatedAt    time.Time
}
