package rabbitmq

import (
	"context"
	"encoding/json"

	"github.com/LgAcerbi/go-video-upload/pkg/rabbitmq"
	"github.com/LgAcerbi/go-video-upload/services/transcode/internal/application/ports"
)

const stepResultQueueName = "upload-process-step"

type stepResultMessage struct {
	UploadID     string `json:"upload_id"`
	VideoID      string `json:"video_id"`
	Step         string `json:"step"`
	Status       string `json:"status"`
	ErrorMessage string `json:"error_message"`
	StoragePath  string `json:"storage_path"`
}

type StepResultPublisher struct {
	conn *rabbitmq.Connection
}

func NewStepResultPublisher(conn *rabbitmq.Connection) ports.StepResultPublisher {
	return &StepResultPublisher{conn: conn}
}

func (p *StepResultPublisher) PublishStepResult(ctx context.Context, uploadID, videoID, step, status, errorMessage, storagePath string) error {
	ch, err := p.conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	if err := rabbitmq.DeclareQueue(ch, stepResultQueueName, true); err != nil {
		return err
	}

	body, err := json.Marshal(stepResultMessage{
		UploadID:     uploadID,
		VideoID:      videoID,
		Step:         step,
		Status:       status,
		ErrorMessage: errorMessage,
		StoragePath:  storagePath,
	})
	if err != nil {
		return err
	}

	return rabbitmq.Publish(ctx, ch, "", stepResultQueueName, body)
}
