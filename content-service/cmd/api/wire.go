//go:build wireinject
// +build wireinject

package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/wire"
	"github.com/septianpadli/talas/content-service/internal/config"
	"github.com/septianpadli/talas/content-service/internal/handler"
	"github.com/septianpadli/talas/content-service/internal/repository"
	"github.com/septianpadli/talas/content-service/internal/usecase"
	"github.com/septianpadli/talas/content-service/pkg/clients"
	"github.com/septianpadli/talas/content-service/pkg/database"
	"github.com/septianpadli/talas/content-service/pkg/logger"
	"github.com/septianpadli/talas/content-service/pkg/media"
	"github.com/septianpadli/talas/content-service/pkg/middleware"
	"github.com/septianpadli/talas/content-service/pkg/rabbitmq"
)

// Ini adalah fungsi yang akan kita panggil di main.go
// Wire akan otomatis mengisi return value-nya (misal *fiber.App)
func InitializeApp() (*fiber.App, error) {
	wire.Build(
		// Config
		config.LoadConfig,

		// Logger
		logger.NewLogger,

		// Middleware
		middleware.NewAuthMiddleware,

		// Database
		database.ConnectDB,

		// Media
		media.NewCloudinaryUploader,
		wire.Bind(new(media.MediaUploader), new(*media.CloudinaryUploader)),

		// Clients
		clients.NewUserClient,

		// RabbitMQ
		rabbitmq.NewRabbitMQPublisher,

		// Repository
		repository.NewShowcaseRepository,

		// Usecase
		usecase.NewShowcaseUsecase,

		// Handler
		handler.NewShowcaseHandler,

		// App
		NewFiberApp,
	)
	
	return &fiber.App{}, nil
}