package rabbitmq

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

func Publish(ctx context.Context, ch *amqp.Channel, exchange, key string, body []byte) error {
	return ch.PublishWithContext(ctx,
		exchange,
		key,
		false,
		false,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)
}

func PublishWithOptions(ctx context.Context, ch *amqp.Channel, exchange, key string, p amqp.Publishing) error {
	if p.DeliveryMode == 0 {
		p.DeliveryMode = amqp.Persistent
	}
	return ch.PublishWithContext(ctx, exchange, key, false, false, p)
}
