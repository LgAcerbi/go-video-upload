package objectstorage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/LgAcerbi/go-video-upload/services/upload/internal/application/ports"
)

type MinIOConfig struct {
	Endpoint         string
	PresignEndpoint  string
	Region           string
	Bucket           string
	AccessKeyID      string
	SecretAccessKey  string
}

type MinIORepository struct {
	client          *s3.Client
	bucket          string
	endpoint        string
	presignEndpoint string
}

func NewMinIORepository(ctx context.Context, cfg MinIOConfig) (*MinIORepository, error) {
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
	return &MinIORepository{
		client:          client,
		bucket:          cfg.Bucket,
		endpoint:        strings.TrimRight(cfg.Endpoint, "/"),
		presignEndpoint: strings.TrimRight(cfg.PresignEndpoint, "/"),
	}, nil
}

func (s *MinIORepository) Upload(ctx context.Context, input *ports.UploadInput) error {
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

func (s *MinIORepository) PresignPut(ctx context.Context, bucket, key string, expiry time.Duration) (string, error) {
	if bucket == "" {
		bucket = s.bucket
	}
	if bucket == "" || key == "" {
		return "", fmt.Errorf("bucket and key are required")
	}
	presignClient := s3.NewPresignClient(s.client)
	req, err := presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expiry
	})
	if err != nil {
		return "", err
	}
	url := req.URL
	if s.presignEndpoint != "" && s.endpoint != "" {
		url = strings.Replace(url, s.endpoint, s.presignEndpoint, 1)
	}
	return url, nil
}
