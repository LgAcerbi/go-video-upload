package ffmpeg

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"github.com/LgAcerbi/go-video-upload/services/transcode/internal/application/ports"
)

type Transcoder struct{}

func NewTranscoder() ports.Transcoder {
	return &Transcoder{}
}

func (t *Transcoder) Transcode(ctx context.Context, inputPath string, height int) (outputPath string, cleanup func(), err error) {
	tmp, err := os.CreateTemp("", "transcode-*.mp4")
	if err != nil {
		return "", nil, fmt.Errorf("create temp file: %w", err)
	}
	outputPath = tmp.Name()
	if err := tmp.Close(); err != nil {
		os.Remove(outputPath)
		return "", nil, err
	}
	cleanup = func() { _ = os.Remove(outputPath) }

	// scale=-2:height keeps aspect ratio (width divisible by 2 for encoder)
	args := []string{
		"-y",
		"-i", inputPath,
		"-vf", fmt.Sprintf("scale=-2:%d", height),
		"-c:v", "libx264",
		"-crf", "23",
		"-preset", "medium",
		"-c:a", "aac",
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

var _ ports.Transcoder = (*Transcoder)(nil)
