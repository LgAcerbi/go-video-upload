package repository

import (
	"context"
	"fmt"

	"github.com/LgAcerbi/go-video-upload/services/upload/internal/ports"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type MinIOConfig struct {
	Endpoint        string
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
}

type MinIOStorage struct {
	client *s3.Client
	bucket string
}

func NewMinIOStorage(ctx context.Context, cfg MinIOConfig) (*MinIOStorage, error) {
	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}
	if cfg.Endpoint == "" || cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" {
		return nil, fmt.Errorf("MinIO requires Endpoint, AccessKeyID, and SecretAccessKey")
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
	return &MinIOStorage{client: client, bucket: cfg.Bucket}, nil
}

func (s *MinIOStorage) Upload(ctx context.Context, input *ports.UploadInput) error {
	if input == nil || input.Body == nil {
		return fmt.Errorf("upload input and body are required")
	}
	bucket := input.Bucket
	if bucket == "" {
		bucket = s.bucket
	}
	if bucket == "" {
		return fmt.Errorf("bucket is required")
	}
	putInput := &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(input.Key),
		Body:        input.Body,
		ContentType: aws.String(input.ContentType),
	}
	if input.ContentLength >= 0 {
		putInput.ContentLength = aws.Int64(input.ContentLength)
	}
	_, err := s.client.PutObject(ctx, putInput)
	return err
}
