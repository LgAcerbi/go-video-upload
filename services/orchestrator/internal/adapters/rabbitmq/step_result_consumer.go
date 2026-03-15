package amqp

import (
	"context"
	"encoding/json"

	"github.com/LgAcerbi/go-video-upload/pkg/logger"
	"github.com/LgAcerbi/go-video-upload/pkg/rabbitmq"
	"github.com/LgAcerbi/go-video-upload/services/orchestrator/internal/application/services"
)

const uploadProcessStepQueueName = "upload-process-step"

type stepResultMessage struct {
	UploadID     string `json:"upload_id"`
	VideoID      string `json:"video_id"`
	Step         string `json:"step"`
	Status       string `json:"status"`
	ErrorMessage string `json:"error_message"`
	StoragePath  string `json:"storage_path"`
}

func RunStepResultConsumer(ctx context.Context, conn *rabbitmq.Connection, svc *service.OrchestratorService, log logger.Logger) error {
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	if err := rabbitmq.DeclareQueue(ch, uploadProcessStepQueueName, true); err != nil {
		return err
	}

	deliveries, err := rabbitmq.Consume(ch, uploadProcessStepQueueName, "orchestrator-step")
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
			var msg stepResultMessage
			if err := json.Unmarshal(d.Body, &msg); err != nil {
				log.Error("invalid upload-process-step message", "error", err, "body", string(d.Body))
				_ = d.Nack(false, false)
				continue
			}
			if err := svc.HandleStepResult(ctx, msg.UploadID, msg.VideoID, msg.Step, msg.Status, msg.ErrorMessage, msg.StoragePath); err != nil {
				log.Error("handle step result failed", "upload_id", msg.UploadID, "step", msg.Step, "error", err)
				_ = d.Nack(false, true)
				continue
			}
			_ = d.Ack(false)
		}
	}
}
