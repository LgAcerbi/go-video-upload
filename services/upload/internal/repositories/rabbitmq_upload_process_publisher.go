package repository

import (
	"context"
	"encoding/json"

	"github.com/LgAcerbi/go-video-upload/pkg/rabbitmq"
	"github.com/LgAcerbi/go-video-upload/services/upload/internal/ports"
)

const uploadProcessQueueName = "upload-process"

type UploadProcessMessage struct {
	VideoID     string `json:"video_id"`
	UploadID    string `json:"upload_id"`
	StoragePath string `json:"storage_path"`
}

type RabbitMQUploadProcessPublisher struct {
	conn *rabbitmq.Connection
}

func NewRabbitMQUploadProcessPublisher(conn *rabbitmq.Connection) ports.UploadProcessPublisher {
	return &RabbitMQUploadProcessPublisher{conn: conn}
}

func (p *RabbitMQUploadProcessPublisher) PublishUploadProcess(ctx context.Context, videoID, uploadID, storagePath string) error {
	ch, err := p.conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	if err := rabbitmq.DeclareQueue(ch, uploadProcessQueueName, true); err != nil {
		return err
	}

	body, err := json.Marshal(UploadProcessMessage{
		VideoID:     videoID,
		UploadID:    uploadID,
		StoragePath: storagePath,
	})
	if err != nil {
		return err
	}

	return rabbitmq.Publish(ctx, ch, "", uploadProcessQueueName, body)
}
