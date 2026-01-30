package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/septianpadli/talas/content-service/internal/config"
	"github.com/sirupsen/logrus"
)

// EventPayload is the standard event structure
type EventPayload struct {
	EventID   string      `json:"event_id"`
	EventType string      `json:"event_type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// EventPublisher defines the interface for publishing events
type EventPublisher interface {
	Publish(ctx context.Context, routingKey string, data interface{}) error
	Close() error
}

// RabbitMQPublisher implements EventPublisher using amqp091-go
type RabbitMQPublisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	cfg     *config.Config
	log     *logrus.Logger
	mu      sync.Mutex
}

// NewRabbitMQPublisher creates a new publisher instance
func NewRabbitMQPublisher(cfg *config.Config, log *logrus.Logger) EventPublisher {
	p := &RabbitMQPublisher{
		cfg: cfg,
		log: log,
	}

	// Attempt initial connection (non-blocking on failure)
	if err := p.connect(); err != nil {
		p.log.Warnf("RabbitMQ: Failed to connect on startup: %v", err)
	}

	return p
}

// connect establishes the connection and channel
func (p *RabbitMQPublisher) connect() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.conn != nil && !p.conn.IsClosed() {
		return nil
	}

	p.log.Infof("RabbitMQ: Connecting to %s", p.cfg.RabbitMQURL)
	conn, err := amqp.Dial(p.cfg.RabbitMQURL)
	if err != nil {
		return fmt.Errorf("failed to dial rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to create channel: %w", err)
	}

	// Declare Exchange (Topic)
	err = ch.ExchangeDeclare(
		"talas.events", // name
		"topic",        // type
		true,           // durable
		false,          // auto-deleted
		false,          // internal
		false,          // no-wait
		nil,            // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	p.conn = conn
	p.channel = ch
	p.log.Info("RabbitMQ: Connected & Exchange 'talas.events' declared")

	return nil
}

// Publish sends an event to the exchange
func (p *RabbitMQPublisher) Publish(ctx context.Context, routingKey string, data interface{}) error {
	// Ensure connection
	if p.channel == nil || p.conn == nil || p.conn.IsClosed() {
		if err := p.connect(); err != nil {
			return fmt.Errorf("rabbitmq not connected: %w", err)
		}
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Create EventPayload
	payload := EventPayload{
		EventID:   uuid.NewString(),
		EventType: routingKey,
		Timestamp: time.Now(),
		Data:      data,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}

	err = p.channel.PublishWithContext(ctx,
		"talas.events", // exchange
		routingKey,     // routing key
		false,          // mandatory
		false,          // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
			Timestamp:   time.Now(),
		},
	)

	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	p.log.Debugf("RabbitMQ: Published event '%s'", routingKey)
	return nil
}

// Close closes the connection and channel
func (p *RabbitMQPublisher) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.channel != nil {
		p.channel.Close()
	}
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}
