package ffprobe

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/LgAcerbi/go-video-upload/services/transcode/internal/application/ports"
)

type stream struct {
	CodecType string `json:"codec_type"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}

type probeOutput struct {
	Streams []stream `json:"streams"`
}

type Prober struct{}

func NewProber() ports.DimensionsProber {
	return &Prober{}
}

func (p *Prober) Probe(ctx context.Context, filePath string) (width, height int, err error) {
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_streams",
		filePath,
	)
	out, err := cmd.Output()
	if err != nil {
		return 0, 0, fmt.Errorf("ffprobe: %w", err)
	}
	var probe probeOutput
	if err := json.Unmarshal(out, &probe); err != nil {
		return 0, 0, fmt.Errorf("parse ffprobe output: %w", err)
	}
	for _, s := range probe.Streams {
		if s.CodecType == "video" {
			return s.Width, s.Height, nil
		}
	}
	return 0, 0, fmt.Errorf("no video stream found")
}

var _ ports.DimensionsProber = (*Prober)(nil)
