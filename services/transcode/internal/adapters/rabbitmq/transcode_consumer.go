package rabbitmq

import (
	"context"
	"encoding/json"

	"github.com/LgAcerbi/go-video-upload/pkg/logger"
	"github.com/LgAcerbi/go-video-upload/pkg/metrics"
	"github.com/LgAcerbi/go-video-upload/pkg/rabbitmq"
	"github.com/LgAcerbi/go-video-upload/services/transcode/internal/application/services"
)

const (
	pipelineStepsExchange = "pipeline-steps"
	transcodeKey          = "transcode"
	transcodeQueueName    = "transcode-transcode"
	serviceTagTranscode   = "transcode"
)

type transcodeMessage struct {
	UploadID string `json:"upload_id"`
}

func RunTranscodeConsumer(ctx context.Context, conn *rabbitmq.Connection, svc *service.TranscodeService, mw *metrics.Writer, log logger.Logger) error {
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	if err := rabbitmq.DeclareExchange(ch, pipelineStepsExchange, rabbitmq.ExchangeTopic, true); err != nil {
		return err
	}
	if err := rabbitmq.DeclareQueue(ch, transcodeQueueName, true); err != nil {
		return err
	}
	if err := rabbitmq.QueueBind(ch, transcodeQueueName, transcodeKey, pipelineStepsExchange); err != nil {
		return err
	}

	deliveries, err := rabbitmq.Consume(ch, transcodeQueueName, "transcode-worker")
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
			var msg transcodeMessage
			if err := json.Unmarshal(d.Body, &msg); err != nil {
				log.Error("invalid transcode message", "error", err, "body", string(d.Body))
				if mw != nil {
					mw.Record("rabbitmq_messages", map[string]string{"service": serviceTagTranscode, "status": "ERROR"}, map[string]interface{}{"input": string(d.Body), "error_message": err.Error()})
				}
				_ = d.Nack(false, false)
				continue
			}
			if err := svc.Transcode(ctx, msg.UploadID); err != nil {
				log.Error("transcode failed", "upload_id", msg.UploadID, "error", err)
				if mw != nil {
					mw.Record("rabbitmq_messages", map[string]string{"service": serviceTagTranscode, "status": "ERROR"}, map[string]interface{}{"input": string(d.Body), "error_message": err.Error()})
				}
				_ = d.Nack(false, false)
				continue
			}
			_ = d.Ack(false)
			if mw != nil {
				mw.Record("rabbitmq_messages", map[string]string{"service": serviceTagTranscode, "status": "OK"}, map[string]interface{}{"input": string(d.Body)})
			}
		}
	}
}
