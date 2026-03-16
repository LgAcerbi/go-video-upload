package amqp

import (
	"context"
	"encoding/json"

	"github.com/LgAcerbi/go-video-upload/pkg/logger"
	"github.com/LgAcerbi/go-video-upload/pkg/rabbitmq"
	"github.com/LgAcerbi/go-video-upload/services/orchestrator/internal/application/services"
)

const uploadProcessQueueName = "upload-process"

type uploadProcessMessage struct {
	VideoID     string `json:"video_id"`
	UploadID    string `json:"upload_id"`
	StoragePath string `json:"storage_path"`
}

func RunUploadProcessConsumer(ctx context.Context, conn *rabbitmq.Connection, svc *service.OrchestratorService, log logger.Logger) error {
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	if err := rabbitmq.DeclareQueue(ch, uploadProcessQueueName, true); err != nil {
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
				_ = d.Nack(false, false)
				continue
			}
			if err := svc.ProcessUploadProcess(ctx, msg.VideoID, msg.UploadID, msg.StoragePath); err != nil {
				log.Error("process upload failed", "upload_id", msg.UploadID, "error", err)
				_ = d.Nack(false, false)
				continue
			}
			_ = d.Ack(false)
		}
	}
}
