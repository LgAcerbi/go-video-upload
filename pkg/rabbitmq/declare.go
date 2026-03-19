package rabbitmq

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	ExchangeDirect  = "direct"
	ExchangeTopic   = "topic"
	ExchangeFanout  = "fanout"
	ExchangeHeaders = "headers"
)

func DeclareExchange(ch *amqp.Channel, name, kind string, durable bool) error {
	return ch.ExchangeDeclare(
		name,
		kind,
		durable,
		false,
		false,
		false,
		nil,
	)
}

func DeclareQueue(ch *amqp.Channel, name string, durable bool) error {
	_, err := ch.QueueDeclare(
		name,
		durable,
		false,
		false,
		false,
		nil,
	)
	return err
}

func DeclareQueueWithArgs(ch *amqp.Channel, name string, durable bool, args amqp.Table) error {
	_, err := ch.QueueDeclare(
		name,
		durable,
		false,
		false,
		false,
		args,
	)
	return err
}

func QueueBind(ch *amqp.Channel, queue, key, exchange string) error {
	return ch.QueueBind(
		queue,
		key,
		exchange,
		false,
		nil,
	)
}
