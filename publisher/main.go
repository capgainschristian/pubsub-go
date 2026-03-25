package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/capgainschristian/pubsub-go/internal/config"
	"github.com/capgainschristian/pubsub-go/internal/events"
	"github.com/capgainschristian/pubsub-go/internal/rabbitmq"
	"go.uber.org/zap"
)

func newID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func main() {
	log, _ := zap.NewProduction()
	defer log.Sync()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config", zap.Error(err))
	}

	rmq, err := rabbitmq.New(cfg.RabbitMQURL, cfg.RabbitMQExchange, log)
	if err != nil {
		log.Fatal("Failed to connect to RabbitMQ", zap.Error(err))
	}

	defer rmq.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Info("Publisher started - publishing sample events every 10s")

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	tick := 0
	publish(ctx, rmq, log, tick)

	for {
		select {
		case <-ctx.Done():
			log.Info("Publisher shutting down")
			return
		case <-ticker.C:
			tick++
			publish(ctx, rmq, log, tick)
		}
	}

}

// showcasing RabbitMQ topic routing to different queues/consumers.
// even = general notification; odd = alert notification
func publish(ctx context.Context, rmq *rabbitmq.Client, log *zap.Logger, tick int) {
	var routingKey, severity, title, body string

	if tick%2 == 0 {
		routingKey = events.RouteNotification
		severity = events.SeverityLow
		title = "Heartbeat"
		body = fmt.Sprintf("Publisher is alive at %s", time.Now().Format(time.RFC3339))
	} else {
		routingKey = events.RouteAlert
		severity = events.SeverityMedium
		title = "Alert"
		body = fmt.Sprintf("Simulated alert fired at %s", time.Now().Format(time.RFC3339))
	}

	evt := events.Event{
		ID:        newID(),
		Type:      "sample",
		Severity:  severity,
		Title:     title,
		Body:      body,
		Source:    "publisher",
		Timestamp: time.Now().UTC(),
		Meta: map[string]string{
			"env":  os.Getenv("APP_ENV"),
			"tick": fmt.Sprintf("%d", tick),
		},
	}

	payload, err := json.Marshal(evt)
	if err != nil {
		log.Error("Failed to marshal event", zap.Error(err))
		return
	}
	if err := rmq.Publish(ctx, routingKey, payload); err != nil {
		log.Error("Failed to publish event", zap.Error(err))
		return
	}

	log.Info("event published",
		zap.String("id", evt.ID),
		zap.String("routing_key", routingKey),
		zap.String("severity", evt.Severity),
	)
}
