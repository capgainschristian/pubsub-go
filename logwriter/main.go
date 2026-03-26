package main

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"syscall"

	"github.com/capgainschristian/pubsub-go/internal/config"
	"github.com/capgainschristian/pubsub-go/internal/dispatch"
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

	dispatchers := []dispatch.EventDispatcher{
		dispatch.NewLogDispatcher(log),
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

			// capture the routing key in case debugging is needed
			log.Debug("processing delivery", zap.String("routing_key", d.RoutingKey))

			if err := dispatch.Dispatch(ctx, evt, dispatchers, log); err != nil {
				log.Error("dispatch failed, nacking message",
					zap.String("event_id", evt.ID),
					zap.Error(err),
				)
				_ = d.Nack(false, false)
				continue
			}

			if err := d.Ack(false); err != nil {
				log.Error("failed to acknowledge message", zap.Error(err))
			}
		}
	}
}
