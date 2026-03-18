package rabbitmq

import (
	"context"
	"encoding/json"

	"github.com/LgAcerbi/go-video-upload/pkg/logger"
	"github.com/LgAcerbi/go-video-upload/pkg/rabbitmq"
	"github.com/LgAcerbi/go-video-upload/services/publish/internal/application/services"
)

const (
	pipelineStepsExchange = "pipeline-steps"
	publishKey            = "publish"
	publishQueueName      = "publish-publish"
)

type publishMessage struct {
	UploadID string `json:"upload_id"`
}

func RunPublishConsumer(ctx context.Context, conn *rabbitmq.Connection, svc *service.PublishService, log logger.Logger) error {
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	if err := rabbitmq.DeclareExchange(ch, pipelineStepsExchange, rabbitmq.ExchangeTopic, true); err != nil {
		return err
	}
	if err := rabbitmq.DeclareQueue(ch, publishQueueName, true); err != nil {
		return err
	}
	if err := rabbitmq.QueueBind(ch, publishQueueName, publishKey, pipelineStepsExchange); err != nil {
		return err
	}

	deliveries, err := rabbitmq.Consume(ch, publishQueueName, "publish-worker")
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
			var msg publishMessage
			if err := json.Unmarshal(d.Body, &msg); err != nil {
				log.Error("invalid publish message", "error", err, "body", string(d.Body))
				_ = d.Nack(false, false)
				continue
			}
			if err := svc.Publish(ctx, msg.UploadID); err != nil {
				log.Error("publish failed", "upload_id", msg.UploadID, "error", err)
				_ = d.Nack(false, false)
				continue
			}
			_ = d.Ack(false)
		}
	}
}
