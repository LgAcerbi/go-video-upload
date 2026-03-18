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

type S3Config struct {
	Endpoint        string
	PresignEndpoint string
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
}

type S3Repository struct {
	client        *s3.Client
	presignClient *s3.Client
	bucket        string
}

func NewS3Repository(ctx context.Context, cfg S3Config) (*S3Repository, error) {
	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}
	var awsCfg aws.Config
	var err error
	if cfg.Endpoint != "" && cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		awsCfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(cfg.Region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				cfg.AccessKeyID, cfg.SecretAccessKey, "",
			)),
		)
		if err != nil {
			return nil, fmt.Errorf("loading AWS config: %w", err)
		}
	} else {
		awsCfg, err = config.LoadDefaultConfig(ctx, config.WithRegion(cfg.Region))
		if err != nil {
			return nil, fmt.Errorf("loading AWS config: %w", err)
		}
	}
	repo := &S3Repository{bucket: cfg.Bucket}
	if cfg.Endpoint != "" {
		endpoint := strings.TrimRight(cfg.Endpoint, "/")
		repo.client = s3.NewFromConfig(awsCfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(endpoint)
			o.UsePathStyle = true
		})
		presignEndpoint := strings.TrimRight(cfg.PresignEndpoint, "/")
		if presignEndpoint != "" {
			repo.presignClient = s3.NewFromConfig(awsCfg, func(o *s3.Options) {
				o.BaseEndpoint = aws.String(presignEndpoint)
				o.UsePathStyle = true
			})
		}
	} else {
		repo.client = s3.NewFromConfig(awsCfg)
	}
	if repo.client == nil {
		repo.client = s3.NewFromConfig(awsCfg)
	}
	return repo, nil
}

func (s *S3Repository) Upload(ctx context.Context, input *ports.UploadInput) error {
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

func (s *S3Repository) PresignPut(ctx context.Context, bucket, key string, expiry time.Duration) (string, error) {
	if bucket == "" {
		bucket = s.bucket
	}
	if bucket == "" || key == "" {
		return "", fmt.Errorf("bucket and key are required")
	}
	client := s.client
	if s.presignClient != nil {
		client = s.presignClient
	}
	presigner := s3.NewPresignClient(client)
	req, err := presigner.PresignPutObject(ctx, &s3.PutObjectInput{
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

func (s *S3Repository) Exists(ctx context.Context, bucket, key string) (bool, error) {
	if bucket == "" {
		bucket = s.bucket
	}
	if bucket == "" || key == "" {
		return false, fmt.Errorf("bucket and key are required")
	}
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "404") || strings.Contains(msg, "not found") || strings.Contains(msg, "nosuchkey") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
