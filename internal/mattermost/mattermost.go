package mattermost

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Webhook payload accepted by Mattermost.
// See https://developers.mattermost.com/integrate/webhooks/incoming/
type Payload struct {
	Text        string       `json:"text"`
	Channel     string       `json:"channel,omitempty"`
	Username    string       `json:"username,omitempty"`
	IconURL     string       `json:"icon_url,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

// Attachment adds rich formatting to a Mattermost message.
type Attachment struct {
	Fallback   string  `json:"fallback"`
	Color      string  `json:"color,omitempty"` // e.g. "#FF0000"
	Title      string  `json:"title,omitempty"`
	Text       string  `json:"text,omitempty"`
	Footer     string  `json:"footer,omitempty"`
	FooterIcon string  `json:"footer_icon,omitempty"`
	Fields     []Field `json:"fields,omitempty"`
}

// Field is a key/value pair rendered inside an Attachment.
type Field struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// Mattermost client
type Client struct {
	webhookURL string
	channel    string
	username   string
	iconURL    string
	httpClient *http.Client
}

func New(webhookURL, channel, username, iconURL string) *Client {
	return &Client{
		webhookURL: webhookURL,
		channel:    channel,
		username:   username,
		iconURL:    iconURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) Send(ctx context.Context, text string) error {
	return c.SendPayload(ctx, Payload{Text: text})
}

func (c *Client) SendPayload(ctx context.Context, p Payload) error {
	if p.Channel == "" {
		p.Channel = c.channel
	}
	if p.Username == "" {
		p.Username = c.username
	}
	if p.IconURL == "" {
		p.IconURL = c.iconURL
	}

	body, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("error marshalling Mattermost payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("building request error: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request to Mattermost: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("non-200 response from Mattermost: %s", resp.Status)
	}

	return nil
}
