package rabbitmq

import (
	"context"
	"encoding/json"

	"github.com/LgAcerbi/go-video-upload/pkg/logger"
	"github.com/LgAcerbi/go-video-upload/pkg/rabbitmq"
	"github.com/LgAcerbi/go-video-upload/services/segment/internal/application/services"
)

const (
	pipelineStepsExchange = "pipeline-steps"
	segmentKey            = "segment"
	segmentQueueName      = "segment-segment"
)

type segmentMessage struct {
	UploadID string `json:"upload_id"`
}

func RunSegmentConsumer(ctx context.Context, conn *rabbitmq.Connection, svc *service.SegmentService, log logger.Logger) error {
	retryCfg := rabbitmq.DefaultRetryConfig()
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	if err := rabbitmq.DeclareExchange(ch, pipelineStepsExchange, rabbitmq.ExchangeTopic, true); err != nil {
		return err
	}
	if err := rabbitmq.DeclareQueue(ch, segmentQueueName, true); err != nil {
		return err
	}
	if err := rabbitmq.DeclareRetryInfrastructure(ch, segmentQueueName, retryCfg); err != nil {
		return err
	}
	if err := rabbitmq.QueueBind(ch, segmentQueueName, segmentKey, pipelineStepsExchange); err != nil {
		return err
	}

	deliveries, err := rabbitmq.Consume(ch, segmentQueueName, "segment-worker")
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case d, ok := <-deliveries:
			if !ok {
				return nil
			}
			var msg segmentMessage
			if err := json.Unmarshal(d.Body, &msg); err != nil {
				log.Error("invalid segment message", "error", err, "body", string(d.Body))
				rabbitmq.SendToDLQ(ctx, ch, d, segmentQueueName, err, log)
				continue
			}
			if err := svc.Segment(ctx, msg.UploadID); err != nil {
				log.Error("segment failed", "upload_id", msg.UploadID, "error", err)
				exhausted := rabbitmq.HandleRetry(ctx, ch, d, segmentQueueName, err, retryCfg, log)
				if exhausted {
					svc.ReportFailed(ctx, msg.UploadID, err)
				}
				continue
			}
			_ = d.Ack(false)
		}
	}
}
