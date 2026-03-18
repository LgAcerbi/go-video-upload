package rabbitmq

import (
	"context"
	"encoding/json"

	"github.com/LgAcerbi/go-video-upload/pkg/logger"
	"github.com/LgAcerbi/go-video-upload/pkg/metrics"
	"github.com/LgAcerbi/go-video-upload/pkg/rabbitmq"
	service "github.com/LgAcerbi/go-video-upload/services/thumbnail/internal/application/services"
)

const (
	pipelineStepsExchange       = "pipeline-steps"
	generateThumbnailKey        = "generate_thumbnail"
	generateThumbnailQueueName  = "thumbnail-generate_thumbnail"
	serviceTagGenerateThumbnail = "generate_thumbnail"
)

type generateThumbnailMessage struct {
	UploadID string `json:"upload_id"`
}

func RunGenerateThumbnailConsumer(ctx context.Context, conn *rabbitmq.Connection, svc *service.ThumbnailService, mw *metrics.Writer, log logger.Logger) error {
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	if err := rabbitmq.DeclareExchange(ch, pipelineStepsExchange, rabbitmq.ExchangeTopic, true); err != nil {
		return err
	}
	if err := rabbitmq.DeclareQueue(ch, generateThumbnailQueueName, true); err != nil {
		return err
	}
	if err := rabbitmq.QueueBind(ch, generateThumbnailQueueName, generateThumbnailKey, pipelineStepsExchange); err != nil {
		return err
	}

	deliveries, err := rabbitmq.Consume(ch, generateThumbnailQueueName, "thumbnail-worker")
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
			var msg generateThumbnailMessage
			if err := json.Unmarshal(d.Body, &msg); err != nil {
				log.Error("invalid generate_thumbnail message", "error", err, "body", string(d.Body))
				if mw != nil {
					mw.Record("rabbitmq_messages", map[string]string{"service": serviceTagGenerateThumbnail, "status": "ERROR"}, map[string]interface{}{"input": string(d.Body), "error_message": err.Error()})
				}
				_ = d.Nack(false, false)
				continue
			}
			if err := svc.GenerateThumbnail(ctx, msg.UploadID); err != nil {
				log.Error("generate thumbnail failed", "upload_id", msg.UploadID, "error", err)
				if mw != nil {
					mw.Record("rabbitmq_messages", map[string]string{"service": serviceTagGenerateThumbnail, "status": "ERROR"}, map[string]interface{}{"input": string(d.Body), "error_message": err.Error()})
				}
				_ = d.Nack(false, false)
				continue
			}
			_ = d.Ack(false)
			if mw != nil {
				mw.Record("rabbitmq_messages", map[string]string{"service": serviceTagGenerateThumbnail, "status": "OK"}, map[string]interface{}{"input": string(d.Body)})
			}
		}
	}
}

