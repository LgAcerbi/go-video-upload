package rabbitmq

import (
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Connection struct {
	conn *amqp.Connection
	mu   sync.Mutex
}

func Connect(cfg *Config) (*Connection, error) {
	if cfg == nil {
		cfg = &Config{}
	}
	conn, err := amqp.Dial(cfg.DSN())
	if err != nil {
		return nil, err
	}
	return &Connection{conn: conn}, nil
}

func (c *Connection) Channel() (*amqp.Channel, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return nil, amqp.ErrClosed
	}
	return c.conn.Channel()
}

func (c *Connection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return nil
	}
	err := c.conn.Close()
	c.conn = nil
	return err
}
