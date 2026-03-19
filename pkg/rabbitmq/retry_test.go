package rabbitmq

import (
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func TestDefaultRetryConfig(t *testing.T) {
	cfg := DefaultRetryConfig()
	if cfg.MaxRetries != 3 {
		t.Fatalf("MaxRetries = %d, want 3", cfg.MaxRetries)
	}
	if len(cfg.Delays) != 3 {
		t.Fatalf("len(Delays) = %d, want 3", len(cfg.Delays))
	}
	if cfg.Delays[0] != time.Second || cfg.Delays[1] != 10*time.Second || cfg.Delays[2] != 30*time.Second {
		t.Fatalf("Delays = %v, want [1s 10s 30s]", cfg.Delays)
	}
	if cfg.DLQTtl != 7*24*time.Hour {
		t.Fatalf("DLQTtl = %v, want 168h", cfg.DLQTtl)
	}
}

func TestGetRetryCount(t *testing.T) {
	tests := []struct {
		name    string
		headers amqp.Table
		want    int
	}{
		{name: "nil headers", headers: nil, want: 0},
		{name: "missing key", headers: amqp.Table{"foo": "bar"}, want: 0},
		{name: "int", headers: amqp.Table{retryHeaderKey: 2}, want: 2},
		{name: "int32", headers: amqp.Table{retryHeaderKey: int32(3)}, want: 3},
		{name: "int64", headers: amqp.Table{retryHeaderKey: int64(4)}, want: 4},
		{name: "float64", headers: amqp.Table{retryHeaderKey: 5.0}, want: 5},
		{name: "string", headers: amqp.Table{retryHeaderKey: "6"}, want: 6},
		{name: "invalid string", headers: amqp.Table{retryHeaderKey: "nope"}, want: 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := getRetryCount(tc.headers)
			if got != tc.want {
				t.Fatalf("getRetryCount() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestDelayQueueName(t *testing.T) {
	got := delayQueueName("upload-process", 2)
	if got != "upload-process.delay.2" {
		t.Fatalf("delayQueueName() = %q, want %q", got, "upload-process.delay.2")
	}
}

func TestCloneHeaders(t *testing.T) {
	src := amqp.Table{"k": "v"}
	cloned := cloneHeaders(src)
	cloned["k"] = "changed"
	if src["k"] == "changed" {
		t.Fatalf("cloneHeaders should return an independent map")
	}
}
