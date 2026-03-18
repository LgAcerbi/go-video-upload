package objectstorage

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/LgAcerbi/go-video-upload/services/thumbnail/internal/application/ports"
)

type ThumbnailUploader struct {
	client *s3.Client
}

func NewThumbnailUploader(ctx context.Context, cfg S3Config) (ports.ThumbnailUploader, error) {
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
	return &ThumbnailUploader{client: client}, nil
}

func (u *ThumbnailUploader) UploadThumbnail(ctx context.Context, bucket, key string, body io.Reader, contentLength int64) error {
	_, err := u.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(bucket),
		Key:           aws.String(key),
		Body:          body,
		ContentLength: aws.Int64(contentLength),
		ContentType:   aws.String("image/jpeg"),
	})
	if err != nil {
		return fmt.Errorf("put object: %w", err)
	}
	return nil
}

var _ ports.ThumbnailUploader = (*ThumbnailUploader)(nil)

