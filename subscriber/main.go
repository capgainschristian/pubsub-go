package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/capgainschristian/pubsub-go/internal/config"
	"github.com/capgainschristian/pubsub-go/internal/dispatch"
	"github.com/capgainschristian/pubsub-go/internal/events"
	"github.com/capgainschristian/pubsub-go/internal/mattermost"
	"github.com/capgainschristian/pubsub-go/internal/rabbitmq"
	"go.uber.org/zap"
)

func main() {
	log, _ := zap.NewProduction()
	defer log.Sync()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("loading config error", zap.Error(err))
	}

	if err := cfg.RequireMattermost(); err != nil {
		log.Fatal("missing required mattermost config", zap.Error(err))
	}

	rmq, err := rabbitmq.New(cfg.RabbitMQURL, cfg.RabbitMQExchange, log)
	if err != nil {
		log.Fatal("creating rabbitmq error", zap.Error(err))
	}
	defer rmq.Close()

	mm := mattermost.New(
		cfg.MattermostWebhookURL,
		cfg.MattermostChannel,
		cfg.MattermostUsername,
		cfg.MattermostIconURL,
	)

	// Wire dispatchers here - add more here as new backends ship.
	dispatchers := []dispatch.EventDispatcher{
		dispatch.NewMattermostDispatcher(mm, 0),
	}

	// Subscribe to every routing key under notification.*
	deliveries, err := rmq.Consume(cfg.RabitMQQueue, events.RouteAll)
	if err != nil {
		log.Fatal("starting consumer error", zap.Error(err))
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Info("Subscriber started - waiting for events", zap.String("queue", cfg.RabitMQQueue), zap.String("binding", events.RouteAll))

	for {
		select {
		case <-ctx.Done():
			log.Info("Subscriber shutting down")
			return
		case d, ok := <-deliveries:
			if !ok {
				log.Warn("Deliveries channel closed, exiting")
				return
			}

			// need to turn this into an interface in the future; apply the adapter pattern
			if err := handle(ctx, d.Body, dispatchers, log); err != nil {
				log.Error("handling delivery error", zap.Error(err))
				// Nack with requeue=false to avoid infinite redelivery of bad messages
				// Allowing dead-letter exchange to handle retries if configured
				_ = d.Nack(false, false)
				continue
			}

			d.Ack(false)
		}
	}
}

func handle(ctx context.Context, body []byte, dispatchers []dispatch.EventDispatcher, log *zap.Logger) error {
	var evt events.Event
	if err := json.Unmarshal(body, &evt); err != nil {
		return fmt.Errorf("error unmarshalling event: %w", err)
	}

	log.Info("Received event", zap.String("id", evt.ID), zap.String("type", evt.Type), zap.String("severity", string(evt.Severity)))

	if err := dispatch.Dispatch(ctx, evt, dispatchers, log); err != nil {
		return fmt.Errorf("error dispatching event: %w", err)
	}

	log.Info("Event dispatched", zap.String("id", evt.ID))
	return nil
}
func init() {
	_ = os.Getenv("APP_ENV") // Just to ensure it's set for the sample event metadata
}
