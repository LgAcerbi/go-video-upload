package rabbitmq

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

func Consume(ch *amqp.Channel, queue, consumerTag string) (<-chan amqp.Delivery, error) {
	return ch.Consume(
		queue,
		consumerTag,
		false,
		false,
		false,
		false,
		nil,
	)
}
