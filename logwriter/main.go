package main

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"syscall"

	"github.com/capgainschristian/pubsub-go/internal/config"
	"github.com/capgainschristian/pubsub-go/internal/events"
	"github.com/capgainschristian/pubsub-go/internal/rabbitmq"
	"go.uber.org/zap"
)

func main() {
	log, _ := zap.NewProduction()
	defer log.Sync()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("loading config", zap.Error(err))
	}

	// Different binding keys per container.
	bindingKey := os.Getenv("LOGWRITER_BINDING")
	if bindingKey == "" {
		bindingKey = events.RouteAll
	}

	rmq, err := rabbitmq.New(cfg.RabbitMQURL, cfg.RabbitMQExchange, log)
	if err != nil {
		log.Fatal("creating rabbitmq error", zap.Error(err))
	}
	defer rmq.Close()

	deliveries, err := rmq.Consume(cfg.RabitMQQueue, bindingKey)
	if err != nil {
		log.Fatal("starting consumer error", zap.Error(err))
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Info("LogWriter started - waiting for events", zap.String("queue", cfg.RabitMQQueue), zap.String("binding", bindingKey))

	for {
		select {
		case <-ctx.Done():
			log.Info("LogWriter shutting down")
			return
		case d, ok := <-deliveries:
			if !ok {
				log.Warn("delivery channel closed, exiting")
				return
			}

			var evt events.Event
			if err := json.Unmarshal(d.Body, &evt); err != nil {
				log.Error("failed to unmarshal event", zap.Error(err))
				_ = d.Nack(false, false)
				continue
			}

			log.Info("EVENT RECEIVED",
				zap.String("id", evt.ID),
				zap.String("routing_key", d.RoutingKey),
				zap.String("type", evt.Type),
				zap.String("severity", evt.Severity),
				zap.String("title", evt.Title),
				zap.String("body", evt.Body),
				zap.String("source", evt.Source),
				zap.Time("timestamp", evt.Timestamp),
				zap.Any("meta", evt.Meta),
			)

			if err := d.Ack(false); err != nil {
				log.Error("failed to acknowledge message", zap.Error(err))
			}
		}
	}
}
