package ffprobe

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/LgAcerbi/go-video-upload/services/metadata/internal/application/ports"
)

type extractor struct{}

func NewExtractor() ports.MetadataExtractor {
	return &extractor{}
}

type ffprobeFormat struct {
	FormatName string `json:"format_name"`
	Duration   string `json:"duration"`
}

type ffprobeStream struct {
	CodecType string `json:"codec_type"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}

type ffprobeOutput struct {
	Format  ffprobeFormat   `json:"format"`
	Streams []ffprobeStream `json:"streams"`
}

func (e *extractor) Extract(ctx context.Context, filePath string) (format string, durationSec float64, width int32, height int32, err error) {
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath,
	)
	out, err := cmd.Output()
	if err != nil {
		return "", 0, 0, 0, fmt.Errorf("ffprobe: %w", err)
	}

	var probe ffprobeOutput
	if err := json.Unmarshal(out, &probe); err != nil {
		return "", 0, 0, 0, fmt.Errorf("parse ffprobe output: %w", err)
	}

	format = parseFormatName(probe.Format.FormatName)
	if probe.Format.Duration != "" {
		durationSec, err = strconv.ParseFloat(probe.Format.Duration, 64)
		if err != nil {
			return format, 0, 0, 0, fmt.Errorf("parse duration: %w", err)
		}
	}
	for _, s := range probe.Streams {
		if s.CodecType == "video" {
			width = int32(s.Width)
			height = int32(s.Height)
			break
		}
	}
	return format, durationSec, width, height, nil
}

func parseFormatName(s string) string {
	parts := strings.SplitN(strings.TrimSpace(s), ",", 2)
	if len(parts) > 0 && parts[0] != "" {
		return strings.TrimSpace(parts[0])
	}
	return s
}

var _ ports.MetadataExtractor = (*extractor)(nil)
