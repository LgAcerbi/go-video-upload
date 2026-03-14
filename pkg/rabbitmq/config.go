package rabbitmq

import (
	"fmt"
	"os"
)

type Config struct {
	URL    string
	Host   string
	Port   string
	User   string
	Pass   string
	Vhost  string
}

func (c *Config) DSN() string {
	if c.URL != "" {
		return c.URL
	}
	host := c.Host
	if host == "" {
		host = "localhost"
	}
	port := c.Port
	if port == "" {
		port = "5672"
	}
	user := c.User
	if user == "" {
		user = "guest"
	}
	pass := c.Pass
	if pass == "" {
		pass = "guest"
	}
	vhost := c.Vhost
	if vhost == "" {
		vhost = "/"
	}
	return fmt.Sprintf("amqp://%s:%s@%s:%s/%s", user, pass, host, port, vhost)
}

func ConfigFromEnv() *Config {
	return &Config{
		URL:   os.Getenv("RABBITMQ_URL"),
		Host:  os.Getenv("RABBITMQ_HOST"),
		Port:  os.Getenv("RABBITMQ_PORT"),
		User:  os.Getenv("RABBITMQ_USER"),
		Pass:  os.Getenv("RABBITMQ_PASS"),
		Vhost: os.Getenv("RABBITMQ_VHOST"),
	}
}
