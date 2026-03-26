package dispatch

import (
	"context"

	"github.com/capgainschristian/pubsub-go/internal/events"
	"go.uber.org/zap"
)

type LogDispatcher struct {
	log *zap.Logger
}

func NewLogDispatcher(log *zap.Logger) *LogDispatcher {
	return &LogDispatcher{log: log}
}

func (l *LogDispatcher) Handle(_ context.Context, evt events.Event) error {
	l.log.Info("EVENT RECEIVED",
		zap.String("id", evt.ID),
		zap.String("type", evt.Type),
		zap.String("severity", evt.Severity),
		zap.String("title", evt.Title),
		zap.String("body", evt.Body),
		zap.String("source", evt.Source),
		zap.Time("timestamp", evt.Timestamp),
		zap.Any("meta", evt.Meta),
	)
	return nil
}
