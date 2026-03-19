package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	pkgrabbitmq "github.com/LgAcerbi/go-video-upload/pkg/rabbitmq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	rabbitUser = "guest"
	rabbitPass = "guest"
)

type RabbitHarness struct {
	Connection *pkgrabbitmq.Connection
	DSN        string
	container  testcontainers.Container
}

func StartRabbitHarness(t *testing.T) *RabbitHarness {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Skipf("skipping integration test: docker runtime unavailable (%v)", r)
		}
	}()

	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "rabbitmq:3.13-alpine",
		ExposedPorts: []string{"5672/tcp"},
		WaitingFor:   wait.ForListeningPort("5672/tcp").WithStartupTimeout(30 * time.Second),
	}
	ctr, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("start rabbitmq container: %v", err)
	}

	host, err := ctr.Host(ctx)
	if err != nil {
		_ = ctr.Terminate(ctx)
		t.Fatalf("resolve rabbitmq host: %v", err)
	}
	mappedPort, err := ctr.MappedPort(ctx, "5672/tcp")
	if err != nil {
		_ = ctr.Terminate(ctx)
		t.Fatalf("resolve rabbitmq mapped port: %v", err)
	}

	dsn := fmt.Sprintf("amqp://%s:%s@%s:%s/", rabbitUser, rabbitPass, host, mappedPort.Port())
	conn, err := pkgrabbitmq.Connect(&pkgrabbitmq.Config{URL: dsn})
	if err != nil {
		_ = ctr.Terminate(ctx)
		t.Fatalf("connect rabbitmq: %v", err)
	}

	return &RabbitHarness{
		Connection: conn,
		DSN:        dsn,
		container:  ctr,
	}
}

func (h *RabbitHarness) Close(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	if h.Connection != nil {
		_ = h.Connection.Close()
	}
	if h.container != nil {
		if err := h.container.Terminate(ctx); err != nil {
			t.Fatalf("terminate rabbitmq container: %v", err)
		}
	}
}
