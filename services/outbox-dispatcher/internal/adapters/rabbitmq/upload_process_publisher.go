package amqp

import (
	"context"

	"github.com/LgAcerbi/go-video-upload/pkg/rabbitmq"
	"github.com/LgAcerbi/go-video-upload/services/outbox-dispatcher/internal/application/ports"
)

const uploadProcessQueueName = "upload-process"

type RabbitMQUploadProcessPublisher struct {
	conn *rabbitmq.Connection
}

func NewRabbitMQUploadProcessPublisher(conn *rabbitmq.Connection) ports.UploadProcessPublisher {
	return &RabbitMQUploadProcessPublisher{conn: conn}
}

func (p *RabbitMQUploadProcessPublisher) PublishUploadProcess(ctx context.Context, payload []byte) error {
	ch, err := p.conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	if err := rabbitmq.DeclareQueue(ch, uploadProcessQueueName, true); err != nil {
		return err
	}
	return rabbitmq.Publish(ctx, ch, "", uploadProcessQueueName, payload)
}
