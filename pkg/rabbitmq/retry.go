package rabbitmq

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/LgAcerbi/go-video-upload/pkg/logger"
	amqp "github.com/rabbitmq/amqp091-go"
)

const retryHeaderKey = "x-retry-count"

type RetryConfig struct {
	MaxRetries int
	Delays     []time.Duration
	DLQTtl     time.Duration
}

func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries: 3,
		Delays:     []time.Duration{time.Second, 10 * time.Second, 30 * time.Second},
		DLQTtl:     7 * 24 * time.Hour,
	}
}

func DeclareRetryInfrastructure(ch *amqp.Channel, queueName string, cfg RetryConfig) error {
	var dlqArgs amqp.Table
	if cfg.DLQTtl > 0 {
		dlqArgs = amqp.Table{
			"x-message-ttl": int32(cfg.DLQTtl / time.Millisecond),
		}
	}
	if err := DeclareQueueWithArgs(ch, queueName+".dlq", true, dlqArgs); err != nil {
		return err
	}

	for idx, d := range cfg.Delays {
		delayQueue := delayQueueName(queueName, idx+1)
		args := amqp.Table{
			"x-message-ttl":             int32(d / time.Millisecond),
			"x-dead-letter-exchange":    "",
			"x-dead-letter-routing-key": queueName,
		}
		if err := DeclareQueueWithArgs(ch, delayQueue, true, args); err != nil {
			return err
		}
	}
	return nil
}

func HandleRetry(ctx context.Context, ch *amqp.Channel, d amqp.Delivery, queueName string, procErr error, cfg RetryConfig, log logger.Logger) bool {
	attempt := getRetryCount(d.Headers)
	nextAttempt := attempt + 1
	if nextAttempt <= cfg.MaxRetries {
		headers := cloneHeaders(d.Headers)
		headers[retryHeaderKey] = int32(nextAttempt)
		delayQueue := delayQueueName(queueName, nextAttempt)
		if err := PublishWithOptions(ctx, ch, "", delayQueue, newPublishing(d, headers)); err != nil {
			log.Error("retry publish failed", "queue", queueName, "retry_attempt", nextAttempt, "error", err)
			_ = d.Nack(false, true)
			return false
		}
		_ = d.Ack(false)
		log.Info("message scheduled for retry", "queue", queueName, "retry_attempt", nextAttempt, "delay", cfg.Delays[nextAttempt-1].String(), "error", procErr.Error())
		return false
	}

	headers := cloneHeaders(d.Headers)
	headers[retryHeaderKey] = int32(attempt)
	headers["x-final-error"] = procErr.Error()
	if err := PublishWithOptions(ctx, ch, "", queueName+".dlq", newPublishing(d, headers)); err != nil {
		log.Error("dlq publish failed", "queue", queueName, "error", err)
		_ = d.Nack(false, true)
		return false
	}
	_ = d.Ack(false)
	log.Error("message moved to dlq after retries exhausted", "queue", queueName, "max_retries", cfg.MaxRetries, "error", procErr.Error())
	return true
}

func SendToDLQ(ctx context.Context, ch *amqp.Channel, d amqp.Delivery, queueName string, procErr error, log logger.Logger) {
	headers := cloneHeaders(d.Headers)
	headers["x-final-error"] = procErr.Error()
	if err := PublishWithOptions(ctx, ch, "", queueName+".dlq", newPublishing(d, headers)); err != nil {
		log.Error("dlq publish failed", "queue", queueName, "error", err)
		_ = d.Nack(false, true)
		return
	}
	_ = d.Ack(false)
	log.Error("message moved directly to dlq", "queue", queueName, "error", procErr.Error())
}

func delayQueueName(queueName string, attempt int) string {
	return fmt.Sprintf("%s.delay.%d", queueName, attempt)
}

func getRetryCount(headers amqp.Table) int {
	if headers == nil {
		return 0
	}
	raw, ok := headers[retryHeaderKey]
	if !ok {
		return 0
	}
	switch v := raw.(type) {
	case int:
		return v
	case int8:
		return int(v)
	case int16:
		return int(v)
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float32:
		return int(v)
	case float64:
		return int(v)
	case string:
		n, err := strconv.Atoi(v)
		if err == nil {
			return n
		}
	}
	return 0
}

func cloneHeaders(src amqp.Table) amqp.Table {
	dst := amqp.Table{}
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func newPublishing(d amqp.Delivery, headers amqp.Table) amqp.Publishing {
	return amqp.Publishing{
		Headers:         headers,
		ContentType:     d.ContentType,
		ContentEncoding: d.ContentEncoding,
		DeliveryMode:    amqp.Persistent,
		Priority:        d.Priority,
		CorrelationId:   d.CorrelationId,
		ReplyTo:         d.ReplyTo,
		Expiration:      d.Expiration,
		MessageId:       d.MessageId,
		Timestamp:       d.Timestamp,
		Type:            d.Type,
		UserId:          d.UserId,
		AppId:           d.AppId,
		Body:            d.Body,
	}
}
