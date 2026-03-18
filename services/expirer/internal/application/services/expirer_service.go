package service

import (
	"context"

	"github.com/LgAcerbi/go-video-upload/services/expirer/internal/application/ports"
)

type ExpirerService struct {
	uploadRepo ports.UploadRepository
}

func NewExpirerService(uploadRepo ports.UploadRepository) *ExpirerService {
	return &ExpirerService{uploadRepo: uploadRepo}
}

type ExpireResult struct {
	Found   int
	Expired int
	Skipped int
}

func (s *ExpirerService) ExpireStaleUploads(ctx context.Context, limit int) (ExpireResult, error) {
	res, err := s.uploadRepo.ExpireStaleUploads(ctx, limit)
	if err != nil {
		return ExpireResult{}, err
	}
	return ExpireResult{Found: res.Found, Expired: res.Expired, Skipped: res.Skipped}, nil
}

