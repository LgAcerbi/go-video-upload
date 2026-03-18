package ffmpeg

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/LgAcerbi/go-video-upload/services/thumbnail/internal/application/ports"
)

type ThumbnailGenerator struct{}

func NewThumbnailGenerator() ports.ThumbnailGenerator {
	return &ThumbnailGenerator{}
}

func (g *ThumbnailGenerator) Generate(ctx context.Context, inputPath string) (outputPath string, cleanup func(), err error) {
	tmp, err := os.CreateTemp("", "thumbnail-*.jpg")
	if err != nil {
		return "", nil, fmt.Errorf("create temp file: %w", err)
	}
	outputPath = tmp.Name()
	if err := tmp.Close(); err != nil {
		_ = os.Remove(outputPath)
		return "", nil, err
	}
	cleanup = func() { _ = os.Remove(outputPath) }

	args := []string{
		"-y",
		"-ss", "1",
		"-i", inputPath,
		"-frames:v", "1",
		"-vf", "scale=320:-1",
		"-q:v", "3",
		outputPath,
	}
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("ffmpeg: %w", err)
	}
	return outputPath, cleanup, nil
}

var _ ports.ThumbnailGenerator = (*ThumbnailGenerator)(nil)

