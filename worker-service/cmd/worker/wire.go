//go:build wireinject
// +build wireinject

package main

import (
	"github.com/google/wire"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/septian/worker-service/internal/config"
	"github.com/septian/worker-service/internal/repository"
	"github.com/septian/worker-service/internal/worker"
	"github.com/septian/worker-service/pkg/infrastructure"
	"github.com/sirupsen/logrus"
)

// WorkerApp holds all dependencies for the worker service
type WorkerApp struct {
	Config   *config.Config
	Logger   *logrus.Logger
	MQConn   *amqp.Connection
	Consumer *worker.ShowcaseConsumer
}

// NewWorkerApp creates a new WorkerApp instance
func NewWorkerApp(
	cfg *config.Config,
	log *logrus.Logger,
	conn *amqp.Connection,
	consumer *worker.ShowcaseConsumer,
) *WorkerApp {
	return &WorkerApp{
		Config:   cfg,
		Logger:   log,
		MQConn:   conn,
		Consumer: consumer,
	}
}

// ProvideLogger provides a configured logrus logger
func ProvideLogger() *logrus.Logger {
	log := logrus.New()
	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetLevel(logrus.InfoLevel)
	return log
}

// ProviderSet defines all providers for Wire
var ProviderSet = wire.NewSet(
	config.LoadConfig,
	ProvideLogger,
	infrastructure.NewRabbitMQConnection,
	infrastructure.NewRabbitMQChannel,
	infrastructure.NewElasticsearchClient,
	repository.NewElasticsearchRepository,
	worker.NewShowcaseConsumer,
	NewWorkerApp,
)

// InitializeWorker is the Wire injector function
func InitializeWorker() (*WorkerApp, error) {
	wire.Build(ProviderSet)
	return nil, nil
}
