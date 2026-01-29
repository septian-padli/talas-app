package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/septianpadli/talas/content-service/internal/handler"
	"github.com/septianpadli/talas/content-service/pkg/middleware"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// NewFiberApp provider to create fiber app instance
// Now depends on ShowcaseHandler AND Logger
func NewFiberApp(
	db *gorm.DB, 
	showcaseHandler *handler.ShowcaseHandler, 
	authMiddleware *middleware.AuthMiddleware, 
	log *logrus.Logger,
) *fiber.App {
	app := fiber.New(fiber.Config{
		BodyLimit: 20 * 1024 * 1024, // 20 MB Limit
	})

	// Default Middlewares
	app.Use(logger.New())
	app.Use(recover.New())

	// CORS
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:3000,http://localhost:3001",
		AllowCredentials: true,
	}))

	// Register Routes
	api := app.Group("/api")

	// Protected Routes (Specific First)
	api.Get("/showcases/me", authMiddleware.Protect, showcaseHandler.GetMyShowcases)

	// Public Routes
	api.Get("/showcases/:slug", showcaseHandler.GetShowcaseBySlug)
	api.Get("/showcases/user/:id", showcaseHandler.GetShowcasesByUser)
	api.Get("/showcases/:id/comments", showcaseHandler.GetShowcaseComments)

	// Protected Group (Generic)
	protected := api.Group("/")
	protected.Use(authMiddleware.Protect)
	protected.Get("/test", func(c *fiber.Ctx) error {
		userID := c.Locals("user_id")
		return c.JSON(fiber.Map{
			"status":  "success",
			"message": "Authenticated!",
			"user_id": userID,
		})
	})

	protected.Post("/showcases", showcaseHandler.CreateShowcase)
	protected.Get("/showcases/:id/collaborators", showcaseHandler.GetCollaborators)
	protected.Post("/showcases/:id/collaborators", showcaseHandler.InviteCollaborators)
	protected.Patch("/showcases/:id", showcaseHandler.UpdateShowcase)
	protected.Delete("/showcases/:id/collaborators/:userId", showcaseHandler.RemoveCollaborator)
	protected.Delete("/showcases/:id", showcaseHandler.DeleteShowcase)
	protected.Post("/showcases/:id/like", showcaseHandler.ToggleLike)
	protected.Post("/showcases/:id/bookmark", showcaseHandler.ToggleBookmark)
	protected.Post("/showcases/:id/comments", showcaseHandler.CreateComment)
	protected.Post("/comments/:id/reply", showcaseHandler.ReplyComment)
	protected.Patch("/comments/:id", showcaseHandler.UpdateComment)
	protected.Delete("/comments/:id", showcaseHandler.DeleteComment)
	protected.Post("/comments/:id/like", showcaseHandler.ToggleCommentLike)

	protected.Delete("/collaborations/invitations/:id", showcaseHandler.DeleteInvitation)
	protected.Get("/collaborations/invitations", showcaseHandler.GetPendingInvitations)
	protected.Patch("/collaborations/:id/response", showcaseHandler.RespondInvitation)

	return app
}
