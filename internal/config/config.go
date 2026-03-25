package config

import (
	"fmt"
	"os"
)

type Config struct {
	// RabbitMQ
	RabbitMQURL      string
	RabbitMQExchange string
	RabitMQQueue     string

	// Mattermost
	MattermostWebhookURL string
	MattermostChannel    string
	MattermostUsername   string
	MattermostIconURL    string

	// App
	LogLevel string
}

// Load reads .env for Config
func Load() (*Config, error) {
	cfg := &Config{
		RabbitMQURL:          getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		RabbitMQExchange:     getEnv("RABBITMQ_EXCHANGE", "events"),
		RabitMQQueue:         getEnv("RABBITMQ_QUEUE", "mattermost.notifications"),
		MattermostWebhookURL: getEnv("MATTERMOST_WEBHOOK_URL", ""),
		MattermostChannel:    getEnv("MATTERMOST_CHANNEL", ""),
		MattermostUsername:   getEnv("MATTERMOST_USERNAME", "PubSub Bot"),
		MattermostIconURL:    getEnv("MATTERMOST_ICON_URL", ""),
		LogLevel:             getEnv("LOG_LEVEL", "info"),
	}

	return cfg, nil
}

func (c *Config) RequireMattermost() error {
	if c.MattermostWebhookURL == "" {
		return fmt.Errorf("MATTERMOST_WEBHOOK_URL is required")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
