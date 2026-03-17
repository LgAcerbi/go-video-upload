package amqp

import (
	"context"
	"encoding/json"

	"github.com/LgAcerbi/go-video-upload/pkg/rabbitmq"
	"github.com/LgAcerbi/go-video-upload/services/upload/internal/application/ports"
)

const uploadProcessQueueName = "upload-process"

type UploadProcessMessage struct {
	UploadID string `json:"upload_id"`
}

type RabbitMQUploadProcessPublisher struct {
	conn *rabbitmq.Connection
}

func NewRabbitMQUploadProcessPublisher(conn *rabbitmq.Connection) ports.UploadProcessPublisher {
	return &RabbitMQUploadProcessPublisher{conn: conn}
}

func (p *RabbitMQUploadProcessPublisher) PublishUploadProcess(ctx context.Context, uploadID string) error {
	ch, err := p.conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	if err := rabbitmq.DeclareQueue(ch, uploadProcessQueueName, true); err != nil {
		return err
	}

	body, err := json.Marshal(UploadProcessMessage{UploadID: uploadID})
	if err != nil {
		return err
	}

	return rabbitmq.Publish(ctx, ch, "", uploadProcessQueueName, body)
}
