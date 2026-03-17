package amqp

import (
	"context"
	"encoding/json"

	"github.com/LgAcerbi/go-video-upload/pkg/rabbitmq"
	"github.com/LgAcerbi/go-video-upload/services/orchestrator/internal/application/ports"
)

const pipelineStepsExchange = "pipeline-steps"

type stepMessage struct {
	UploadID string `json:"upload_id"`
}

type StepPublisher struct {
	conn *rabbitmq.Connection
}

func NewStepPublisher(conn *rabbitmq.Connection) ports.StepPublisher {
	return &StepPublisher{conn: conn}
}

func (p *StepPublisher) PublishStep(ctx context.Context, step, uploadID string) error {
	ch, err := p.conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	if err := rabbitmq.DeclareExchange(ch, pipelineStepsExchange, rabbitmq.ExchangeTopic, true); err != nil {
		return err
	}

	body, err := json.Marshal(stepMessage{UploadID: uploadID})
	if err != nil {
		return err
	}

	return rabbitmq.Publish(ctx, ch, pipelineStepsExchange, step, body)
}
