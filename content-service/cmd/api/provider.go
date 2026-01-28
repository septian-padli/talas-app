package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/septianpadli/talas/content-service/internal/handler"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// NewFiberApp provider to create fiber app instance
// Now depends on ProjectHandler AND Logger
func NewFiberApp(db *gorm.DB, projectHandler *handler.ProjectHandler, log *logrus.Logger) *fiber.App {
	app := fiber.New()

	// Default Middlewares
	app.Use(logger.New())
	app.Use(recover.New())

	// Register Routes
	api := app.Group("/api")
	api.Get("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "success",
			"message": "Content Service with Clean Arch is working!",
		})
	})

	// Project Routes
	api.Post("/projects", projectHandler.CreateProject)

	return app
}
