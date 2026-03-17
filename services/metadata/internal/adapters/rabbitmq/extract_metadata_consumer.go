package amqp

import (
	"context"
	"encoding/json"

	"github.com/LgAcerbi/go-video-upload/pkg/logger"
	"github.com/LgAcerbi/go-video-upload/pkg/metrics"
	"github.com/LgAcerbi/go-video-upload/pkg/rabbitmq"
	"github.com/LgAcerbi/go-video-upload/services/metadata/internal/application/services"
)

const (
	pipelineStepsExchange = "pipeline-steps"
	extractMetadataKey    = "extract_metadata"
	metadataQueueName     = "metadata-extract_metadata"
	serviceTagExtractMeta = "extract_metadata"
)

type stepMessage struct {
	UploadID string `json:"upload_id"`
}

func RunExtractMetadataConsumer(ctx context.Context, conn *rabbitmq.Connection, svc *service.MetadataService, mw *metrics.Writer, log logger.Logger) error {
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	if err := rabbitmq.DeclareExchange(ch, pipelineStepsExchange, rabbitmq.ExchangeTopic, true); err != nil {
		return err
	}
	if err := rabbitmq.DeclareQueue(ch, metadataQueueName, true); err != nil {
		return err
	}
	if err := rabbitmq.QueueBind(ch, metadataQueueName, extractMetadataKey, pipelineStepsExchange); err != nil {
		return err
	}

	deliveries, err := rabbitmq.Consume(ch, metadataQueueName, "metadata-extract")
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
			var msg stepMessage
			if err := json.Unmarshal(d.Body, &msg); err != nil {
				log.Error("invalid extract_metadata message", "error", err, "body", string(d.Body))
				if mw != nil {
					mw.Record("rabbitmq_messages", map[string]string{"service": serviceTagExtractMeta, "status": "ERROR"}, map[string]interface{}{"input": string(d.Body), "error_message": err.Error()})
				}
				_ = d.Nack(false, false)
				continue
			}
			if err := svc.ExtractMetadata(ctx, msg.UploadID); err != nil {
				log.Error("extract metadata failed", "upload_id", msg.UploadID, "error", err)
				if mw != nil {
					mw.Record("rabbitmq_messages", map[string]string{"service": serviceTagExtractMeta, "status": "ERROR"}, map[string]interface{}{"input": string(d.Body), "error_message": err.Error()})
				}
				_ = d.Nack(false, false)
				continue
			}
			_ = d.Ack(false)
			if mw != nil {
				mw.Record("rabbitmq_messages", map[string]string{"service": serviceTagExtractMeta, "status": "OK"}, map[string]interface{}{"input": string(d.Body)})
			}
		}
	}
}
