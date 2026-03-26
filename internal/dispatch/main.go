package dispatch

import (
	"context"
	"errors"
	"fmt"

	"github.com/capgainschristian/pubsub-go/internal/events"
	"go.uber.org/zap"
)

type EventDispatcher interface {
	Handle(ctx context.Context, evt events.Event) error
}

func Dispatch(ctx context.Context, evt events.Event, dispatchers []EventDispatcher, log *zap.Logger) error {
	var errs []error
	for _, d := range dispatchers {
		if err := d.Handle(ctx, evt); err != nil {
			log.Error("dispatch failed",
				zap.String("event_id", evt.ID),
				zap.Error(err),
			)
			errs = append(errs, fmt.Errorf("dispatcher error: %w", err))
		}
	}
	return errors.Join(errs...)
}
