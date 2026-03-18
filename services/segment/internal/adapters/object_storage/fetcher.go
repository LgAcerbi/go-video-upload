package objectstorage

import (
	"context"
	"fmt"
	"io"
	"os"

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

type VideoFileFetcher struct {
	client *s3.Client
}

func NewVideoFileFetcher(ctx context.Context, cfg S3Config) (*VideoFileFetcher, error) {
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
	return &VideoFileFetcher{client: client}, nil
}

func (f *VideoFileFetcher) FetchToTempFile(ctx context.Context, bucket, key string) (path string, cleanup func(), err error) {
	out, err := f.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return "", nil, fmt.Errorf("get object: %w", err)
	}
	defer out.Body.Close()

	tmp, err := os.CreateTemp("", "segment-*.mp4")
	if err != nil {
		return "", nil, fmt.Errorf("create temp file: %w", err)
	}
	path = tmp.Name()
	cleanup = func() { _ = os.Remove(path) }

	if _, err := io.Copy(tmp, out.Body); err != nil {
		tmp.Close()
		cleanup()
		return "", nil, fmt.Errorf("copy to temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return "", nil, err
	}
	return path, cleanup, nil
}

var _ ports.VideoFileFetcher = (*VideoFileFetcher)(nil)
