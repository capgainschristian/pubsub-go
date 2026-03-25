package rabbitmq

import (
	"context"
	"fmt"
	"time"

	ampq "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

const (
	reconnectDelay = 5 * time.Second
	maxRetries     = 10
)

// Wrapper for RabbitMQ client
type Client struct {
	url      string
	exchange string
	conn     *ampq.Connection
	ch       *ampq.Channel
	log      *zap.Logger
}

// Setup a new client; declares the topic exchange
func New(url, exchange string, log *zap.Logger) (*Client, error) {
	c := &Client{url: url, exchange: exchange, log: log}
	if err := c.connect(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Client) connect() error {
	var err error
	for i := 0; i < maxRetries; i++ {
		c.log.Info("Connecting to RabbitMQ",
			zap.String("url", c.url),
			zap.Int("attempt", i+1),
		)
		c.conn, err = ampq.Dial(c.url)
		if err == nil {
			break
		}
		c.log.Warn("RabbitMQ not ready, retrying..",
			zap.Error(err),
			zap.Duration("delay", reconnectDelay),
		)
		time.Sleep(reconnectDelay)
	}
	if err != nil {
		return fmt.Errorf("exhausted retries connecting to RabbitMQ: %w", err)
	}

	c.ch, err = c.conn.Channel()
	if err != nil {
		return fmt.Errorf("opening channel error: %w", err)
	}

	return c.ch.ExchangeDeclare(
		c.exchange, //name
		"topic",    // kind
		true,       // durable
		false,      // autoDelete
		false,      // internal
		false,      // no-wait
		nil,        // args
	)
}

func (c *Client) Publish(ctx context.Context, routingKey string, body []byte) error {
	return c.ch.PublishWithContext(
		ctx,
		c.exchange,
		routingKey,
		false, // mandatory
		false, // immediate
		ampq.Publishing{
			ContentType:  "application/json",
			DeliveryMode: ampq.Persistent,
			Timestamp:    time.Now(),
			Body:         body,
		},
	)
}

func (c *Client) Consume(queue, bindingKey string) (<-chan ampq.Delivery, error) {
	q, err := c.ch.QueueDeclare(
		queue, //name
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return nil, fmt.Errorf("declaring queue %q error: %w", queue, err)
	}

	if err := c.ch.QueueBind(q.Name, bindingKey, c.exchange, false, nil); err != nil {
		return nil, fmt.Errorf("binding queue %q to exchange %q error: %w", queue, c.exchange, err)
	}

	// Don't hand more than one message to a worker at a time
	if err := c.ch.Qos(1, 0, false); err != nil {
		return nil, fmt.Errorf("setting QoS error: %w", err)
	}

	return c.ch.Consume(
		q.Name, // queue
		"",     // consumer tag
		false,  // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
}

func (c *Client) Close() {
	if c.ch != nil {
		c.ch.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
	c.log.Info("RabbitMQ connection closed")
}
