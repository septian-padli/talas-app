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
	"github.com/septianpadli/talas/content-service/pkg/database"
	"github.com/septianpadli/talas/content-service/pkg/logger"
	// Import package internal kamu nanti disini
	// "github.com/septianpadli/talas/content-service/internal/repository"
	// "github.com/septianpadli/talas/content-service/internal/usecase"
	// "github.com/septianpadli/talas/content-service/internal/handler"
)

// Ini adalah fungsi yang akan kita panggil di main.go
// Wire akan otomatis mengisi return value-nya (misal *fiber.App)
func InitializeApp() (*fiber.App, error) {
	wire.Build(
		// Config
		config.LoadConfig,

		// Logger
		logger.NewLogger,

		// Database
		database.ConnectDB,

		// Repository
		repository.NewProjectRepository,

		// Usecase
		usecase.NewProjectUsecase,

		// Handler
		handler.NewProjectHandler,

		// App
		NewFiberApp,
	)
	
	return &fiber.App{}, nil
}