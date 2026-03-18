package ffmpeg

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/LgAcerbi/go-video-upload/services/segment/internal/application/ports"
)

type S3Config struct {
	Endpoint        string
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
}

type HlsProducer struct {
	client *s3.Client
}

func NewHlsProducer(ctx context.Context, cfg S3Config) (*HlsProducer, error) {
	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}
	if cfg.Endpoint == "" || cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" {
		return nil, fmt.Errorf("S3 requires Endpoint, AccessKeyID, and SecretAccessKey")
	}
	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID, cfg.SecretAccessKey, "",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(cfg.Endpoint)
		o.UsePathStyle = true
	})
	return &HlsProducer{client: client}, nil
}

func (p *HlsProducer) ProduceAndUpload(ctx context.Context, bucket, outputPrefix, localMp4Path string) error {
	dir, err := os.MkdirTemp("", "hls-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	playlistPath := filepath.Join(dir, "playlist.m3u8")
	segmentPattern := filepath.Join(dir, "seg%03d.ts")

	// ffmpeg -i input.mp4 -hls_time 6 -hls_playlist_type vod -hls_segment_filename seg%03d.ts -f hls playlist.m3u8
	args := []string{
		"-y",
		"-i", localMp4Path,
		"-hls_time", "6",
		"-hls_playlist_type", "vod",
		"-hls_segment_filename", segmentPattern,
		"-f", "hls",
		playlistPath,
	}
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg: %w (output: %s)", err, string(out))
	}

	// Upload all files in dir to S3 with prefix (use forward slashes for S3)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read dir: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		fpath := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(fpath)
		if err != nil {
			return fmt.Errorf("read %s: %w", e.Name(), err)
		}
		key := outputPrefix + "/" + e.Name()
		contentType := "application/vnd.apple.mpegurl"
		if filepath.Ext(e.Name()) == ".ts" {
			contentType = "video/mp2t"
		}
		_, err = p.client.PutObject(ctx, &s3.PutObjectInput{
			Bucket:      aws.String(bucket),
			Key:         aws.String(key),
			Body:        bytes.NewReader(data),
			ContentType: aws.String(contentType),
		})
		if err != nil {
			return fmt.Errorf("put %s: %w", key, err)
		}
	}
	return nil
}

var _ ports.HlsProducer = (*HlsProducer)(nil)
