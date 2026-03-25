package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/capgainschristian/pubsub-go/internal/config"
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
			if err := handle(ctx, d.Body, mm, log); err != nil {
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

func handle(ctx context.Context, body []byte, mm *mattermost.Client, log *zap.Logger) error {
	var evt events.Event
	if err := json.Unmarshal(body, &evt); err != nil {
		return fmt.Errorf("error unmarshalling event: %w", err)
	}

	log.Info("Received event", zap.String("id", evt.ID), zap.String("type", evt.Type), zap.String("severity", string(evt.Severity)))

	payload := buildMattermostPayload(evt)

	postCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	if err := mm.SendPayload(postCtx, payload); err != nil {
		return fmt.Errorf("error sending to Mattermost: %w", err)
	}

	log.Info("Event sent to Mattermost", zap.String("id", evt.ID))
	return nil
}

func buildMattermostPayload(evt events.Event) mattermost.Payload {
	fields := []mattermost.Field{
		{Title: "Source", Value: evt.Source, Short: true},
		{Title: "Severity", Value: evt.Severity, Short: true},
		{Title: "Event ID", Value: evt.ID, Short: false},
	}

	for k, v := range evt.Meta {
		fields = append(fields, mattermost.Field{Title: k, Value: v, Short: true})
	}

	return mattermost.Payload{
		Attachments: []mattermost.Attachment{
			{
				Fallback: fmt.Sprintf("[%s] %s", evt.Severity, evt.Title),
				Color:    events.SeverityColor(string(evt.Severity)),
				Title:    evt.Title,
				Text:     evt.Body,
				Footer:   "pubsub-go - " + evt.Timestamp.Format(time.RFC3339),
				Fields:   fields,
			},
		},
	}
}

func init() {
	_ = os.Getenv("APP_ENV") // Just to ensure it's set for the sample event metadata
}
