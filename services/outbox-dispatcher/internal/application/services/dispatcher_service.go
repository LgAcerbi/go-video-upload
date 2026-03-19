package service

import (
	"context"
	"fmt"
	"time"

	"github.com/LgAcerbi/go-video-upload/services/outbox-dispatcher/internal/application/ports"
)

const uploadProcessStartEventType = "upload_process_start"

type DispatcherService struct {
	outboxRepo       ports.OutboxRepository
	uploadProcessPub ports.UploadProcessPublisher
}

func NewDispatcherService(outboxRepo ports.OutboxRepository, uploadProcessPub ports.UploadProcessPublisher) *DispatcherService {
	return &DispatcherService{
		outboxRepo:       outboxRepo,
		uploadProcessPub: uploadProcessPub,
	}
}

type DispatchResult struct {
	Claimed int
	Sent    int
	Retried int
	Skipped int
}

func (s *DispatcherService) DispatchPending(ctx context.Context, limit int) (DispatchResult, error) {
	events, err := s.outboxRepo.ClaimPendingBatch(ctx, limit)
	if err != nil {
		return DispatchResult{}, err
	}

	res := DispatchResult{Claimed: len(events)}
	for _, event := range events {
		switch event.EventType {
		case uploadProcessStartEventType:
			if err := s.uploadProcessPub.PublishUploadProcess(ctx, event.Payload); err != nil {
				retryAfter := retryAfterSeconds(event.Attempts)
				if markErr := s.outboxRepo.MarkPendingWithError(ctx, event.ID, err.Error(), retryAfter); markErr != nil {
					return res, fmt.Errorf("publish failed (%w) and mark retry failed (%v)", err, markErr)
				}
				res.Retried++
				continue
			}
		default:
			if err := s.outboxRepo.MarkSent(ctx, event.ID); err != nil {
				return res, err
			}
			res.Skipped++
			continue
		}

		if err := s.outboxRepo.MarkSent(ctx, event.ID); err != nil {
			return res, err
		}
		res.Sent++
	}

	return res, nil
}

func retryAfterSeconds(attempts int) int {
	if attempts < 1 {
		attempts = 1
	}
	delay := time.Second * time.Duration(1<<min(attempts-1, 8))
	return int(delay / time.Second)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
