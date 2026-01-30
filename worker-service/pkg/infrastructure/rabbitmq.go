package infrastructure

import (
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/septian/worker-service/internal/config"
	"github.com/sirupsen/logrus"
)

func NewRabbitMQConnection(cfg *config.Config, log *logrus.Logger) (*amqp.Connection, error) {
	var conn *amqp.Connection
	var err error

	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		conn, err = amqp.Dial(cfg.RabbitMQURL)
		if err == nil {
			log.Info("Successfully connected to RabbitMQ")
			return conn, nil
		}
		log.Warnf("RabbitMQ connection attempt %d/%d failed: %v", i+1, maxRetries, err)
		time.Sleep(2 * time.Second)
	}

	return nil, fmt.Errorf("failed to connect to RabbitMQ after %d retries: %w", maxRetries, err)
}

func NewRabbitMQChannel(conn *amqp.Connection) (*amqp.Channel, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open RabbitMQ channel: %w", err)
	}
	return ch, nil
}
