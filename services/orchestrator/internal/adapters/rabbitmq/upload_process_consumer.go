package amqp

import (
	"context"
	"encoding/json"

	"github.com/LgAcerbi/go-video-upload/pkg/logger"
	"github.com/LgAcerbi/go-video-upload/pkg/metrics"
	"github.com/LgAcerbi/go-video-upload/pkg/rabbitmq"
	"github.com/LgAcerbi/go-video-upload/services/orchestrator/internal/application/services"
)

const uploadProcessQueueName = "upload-process"
const serviceTagUploadProcess = "upload-process"

type uploadProcessMessage struct {
	UploadID string `json:"upload_id"`
}

func RunUploadProcessConsumer(ctx context.Context, conn *rabbitmq.Connection, svc *service.OrchestratorService, mw *metrics.Writer, log logger.Logger) error {
	retryCfg := rabbitmq.DefaultRetryConfig()
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	if err := rabbitmq.DeclareQueue(ch, uploadProcessQueueName, true); err != nil {
		return err
	}
	if err := rabbitmq.DeclareRetryInfrastructure(ch, uploadProcessQueueName, retryCfg); err != nil {
		return err
	}

	deliveries, err := rabbitmq.Consume(ch, uploadProcessQueueName, "orchestrator")
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
			var msg uploadProcessMessage
			if err := json.Unmarshal(d.Body, &msg); err != nil {
				log.Error("invalid upload-process message", "error", err, "body", string(d.Body))
				if mw != nil {
					mw.Record("rabbitmq_messages", map[string]string{"service": serviceTagUploadProcess, "status": "ERROR"}, map[string]interface{}{"input": string(d.Body), "error_message": err.Error()})
				}
				rabbitmq.SendToDLQ(ctx, ch, d, uploadProcessQueueName, err, log)
				continue
			}
			if err := svc.ProcessUploadProcess(ctx, msg.UploadID); err != nil {
				log.Error("process upload failed", "upload_id", msg.UploadID, "error", err)
				if mw != nil {
					mw.Record("rabbitmq_messages", map[string]string{"service": serviceTagUploadProcess, "status": "ERROR"}, map[string]interface{}{"input": string(d.Body), "error_message": err.Error()})
				}
				_ = rabbitmq.HandleRetry(ctx, ch, d, uploadProcessQueueName, err, retryCfg, log)
				continue
			}
			_ = d.Ack(false)
			if mw != nil {
				mw.Record("rabbitmq_messages", map[string]string{"service": serviceTagUploadProcess, "status": "OK"}, map[string]interface{}{"input": string(d.Body)})
			}
		}
	}
}
