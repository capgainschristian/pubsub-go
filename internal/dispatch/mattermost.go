package dispatch

import (
	"context"
	"fmt"
	"time"

	"github.com/capgainschristian/pubsub-go/internal/events"
	"github.com/capgainschristian/pubsub-go/internal/mattermost"
)

type MattermostDispatcher struct {
	client  *mattermost.Client
	timeout time.Duration
}

func NewMattermostDispatcher(client *mattermost.Client, timeout time.Duration) *MattermostDispatcher {
	if timeout == 0 {
		timeout = 15 * time.Second
	}
	return &MattermostDispatcher{client: client, timeout: timeout}
}

func (m *MattermostDispatcher) Handle(ctx context.Context, evt events.Event) error {
	payload := buildMattermostPayload(evt)

	postCtx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	if err := m.client.SendPayload(postCtx, payload); err != nil {
		return fmt.Errorf("mattermost send payload error: %w", err)
	}
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
				Color:    events.SeverityColor(evt.Severity),
				Title:    evt.Title,
				Text:     evt.Body,
				Footer:   "pubsub-go - " + evt.Timestamp.Format(time.RFC3339),
				Fields:   fields,
			},
		},
	}
}
