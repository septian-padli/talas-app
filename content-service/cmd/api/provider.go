package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/redis/go-redis/v9"
	"github.com/septianpadli/talas/content-service/internal/config"
	"github.com/septianpadli/talas/content-service/internal/handler"
	internalMiddleware "github.com/septianpadli/talas/content-service/internal/middleware"
	"github.com/septianpadli/talas/content-service/pkg/middleware"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const showcaseIDRoute = "/showcases/:id"

// NewFiberApp provider to create fiber app instance
// Now depends on ShowcaseHandler AND Logger
func NewFiberApp(
	db *gorm.DB,
	redisClient *redis.Client,
	cfg *config.Config,
	showcaseHandler *handler.ShowcaseHandler,
	internalShowcaseHandler *handler.InternalShowcaseHandler,
	authMiddleware *middleware.AuthMiddleware,
	log *logrus.Logger,
) *fiber.App {
	app := fiber.New(fiber.Config{
		BodyLimit: 20 * 1024 * 1024, // 20 MB Limit
	})

	// Default Middlewares
	app.Use(recover.New())
	app.Use(requestid.New())
	app.Use(internalMiddleware.NewRequestLogger(log))

	// CORS
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:3000,http://localhost:3001",
		AllowCredentials: true,
	}))

	// Register Routes
	api := app.Group("/api")

	// Internal Routes (MUST BE FIRST - Protected by Shared Secret)
	internal := api.Group("/internal")
	internal.Use(internalMiddleware.InternalAuthMiddleware(cfg))
	internal.Get(showcaseIDRoute, internalShowcaseHandler.GetShowcaseInternal)

	// Protected Routes (Specific First)
	api.Get("/showcases/me", authMiddleware.Protect, showcaseHandler.GetMyShowcases)

	// Public Routes
	api.Get("/search", showcaseHandler.SearchShowcases)
	api.Get("/feeds/trending", showcaseHandler.GetTrendingFeeds) // Trending Feeds
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
	protected.Get(showcaseIDRoute+"/collaborators", showcaseHandler.GetCollaborators)
	protected.Post(showcaseIDRoute+"/collaborators", showcaseHandler.InviteCollaborators)
	protected.Patch(showcaseIDRoute, showcaseHandler.UpdateShowcase)
	protected.Delete(showcaseIDRoute+"/collaborators/:userId", showcaseHandler.RemoveCollaborator)
	protected.Delete(showcaseIDRoute, showcaseHandler.DeleteShowcase)
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
