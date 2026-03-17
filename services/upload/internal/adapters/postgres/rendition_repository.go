package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/LgAcerbi/go-video-upload/pkg/util"
	"github.com/LgAcerbi/go-video-upload/services/upload/internal/application/ports"
	"github.com/LgAcerbi/go-video-upload/services/upload/internal/domain/entities"
)

type RenditionRepository struct {
	pool *pgxpool.Pool
}

func NewRenditionRepository(pool *pgxpool.Pool) ports.RenditionRepository {
	return &RenditionRepository{pool: pool}
}

func (r *RenditionRepository) CreateBatch(ctx context.Context, videoID, originalStoragePath string, originalWidth, originalHeight int, targetHeights []int) error {
	// Insert original rendition (ready)
	orig := &entities.Rendition{
		VideoID:     videoID,
		Resolution:  entities.ResolutionOriginal,
		StoragePath: &originalStoragePath,
		Width:       ptrInt(originalWidth),
		Height:      ptrInt(originalHeight),
		Status:      entities.RenditionStatusReady,
	}
	if err := r.insertOne(ctx, orig); err != nil {
		return err
	}
	// Insert pending target renditions (width from aspect ratio so dimensions are set at creation)
	for _, h := range targetHeights {
		resolution := fmt.Sprintf("%dp", h)
		width := scaleWidthToEven(originalWidth, originalHeight, h)
		rend := &entities.Rendition{
			VideoID:     videoID,
			Resolution:  resolution,
			StoragePath: nil,
			Width:       ptrInt(width),
			Height:      ptrInt(h),
			Status:      entities.RenditionStatusPending,
		}
		if err := r.insertOne(ctx, rend); err != nil {
			return err
		}
	}
	return nil
}

func ptrInt(n int) *int { return &n }

// scaleWidthToEven returns width for target height preserving aspect ratio, rounded to even (matches ffmpeg scale=-2:H).
func scaleWidthToEven(origW, origH, targetH int) int {
	if origH <= 0 {
		return 0
	}
	w := (origW * targetH) / origH
	if w&1 != 0 {
		w--
	}
	return w
}

func (r *RenditionRepository) insertOne(ctx context.Context, rend *entities.Rendition) error {
	query := `
		INSERT INTO video_renditions (id, video_id, resolution, storage_path, format, width, height, bitrate_kbps, status, created_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, NOW())`
	_, err := r.pool.Exec(ctx, query,
		rend.VideoID, rend.Resolution, rend.StoragePath, util.NullIfEmpty(rend.Format), rend.Width, rend.Height, rend.BitrateKbps, rend.Status)
	return err
}

func (r *RenditionRepository) ListPendingByVideoID(ctx context.Context, videoID string) ([]*entities.Rendition, error) {
	query := `
		SELECT id, video_id, resolution, storage_path, COALESCE(format, ''), width, height, bitrate_kbps, status, created_at
		FROM video_renditions
		WHERE video_id = $1 AND status = $2
		ORDER BY height DESC`
	rows, err := r.pool.Query(ctx, query, videoID, entities.RenditionStatusPending)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*entities.Rendition
	for rows.Next() {
		var rend entities.Rendition
		var storagePath *string
		if err := rows.Scan(&rend.ID, &rend.VideoID, &rend.Resolution, &storagePath, &rend.Format, &rend.Width, &rend.Height, &rend.BitrateKbps, &rend.Status, &rend.CreatedAt); err != nil {
			return nil, err
		}
		rend.StoragePath = storagePath
		out = append(out, &rend)
	}
	return out, rows.Err()
}

func (r *RenditionRepository) Update(ctx context.Context, rend *entities.Rendition) error {
	query := `
		UPDATE video_renditions
		SET storage_path = $2, status = $3, width = $4, height = $5, bitrate_kbps = $6
		WHERE video_id = $1 AND resolution = $7`
	_, err := r.pool.Exec(ctx, query,
		rend.VideoID, rend.StoragePath, rend.Status, rend.Width, rend.Height, rend.BitrateKbps, rend.Resolution)
	return err
}

var _ ports.RenditionRepository = (*RenditionRepository)(nil)
