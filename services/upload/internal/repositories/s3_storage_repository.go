package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/LgAcerbi/go-video-upload/services/upload/internal/ports"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Config struct {
	Region string
	Bucket string
}

type S3StorageRepository struct {
	client *s3.Client
	bucket string
}

func NewS3StorageRepository(ctx context.Context, cfg S3Config) (*S3StorageRepository, error) {
	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}
	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(cfg.Region))
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}
	client := s3.NewFromConfig(awsCfg)
	return &S3StorageRepository{client: client, bucket: cfg.Bucket}, nil
}

func (s *S3StorageRepository) Upload(ctx context.Context, input *ports.UploadInput) error {
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

func (s *S3StorageRepository) PresignPut(ctx context.Context, bucket, key string, expiry time.Duration) (string, error) {
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
	return req.URL, nil
}
