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

// HandlerGroup to group all handlers
type HandlerGroup struct {
	ShowcaseHandler         *handler.ShowcaseHandler
	InternalShowcaseHandler *handler.InternalShowcaseHandler
	CategoryHandler         *handler.CategoryHandler
}

func NewHandlerGroup(
	showcaseHandler *handler.ShowcaseHandler,
	internalShowcaseHandler *handler.InternalShowcaseHandler,
	categoryHandler *handler.CategoryHandler,
) *HandlerGroup {
	return &HandlerGroup{
		ShowcaseHandler:         showcaseHandler,
		InternalShowcaseHandler: internalShowcaseHandler,
		CategoryHandler:         categoryHandler,
	}
}

// NewFiberApp provider to create fiber app instance
func NewFiberApp(
	db *gorm.DB,
	redisClient *redis.Client,
	cfg *config.Config,
	handlers *HandlerGroup,
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
	internal.Get(showcaseIDRoute, handlers.InternalShowcaseHandler.GetShowcaseInternal)

	// Protected Routes (Specific First)
	api.Get("/showcases/me", authMiddleware.Protect, handlers.ShowcaseHandler.GetMyShowcases)

	// Public Routes
	api.Get("/categories", handlers.CategoryHandler.GetCategories)
	api.Get("/categories/:id", handlers.CategoryHandler.GetCategoryDetail)
	api.Get("/search", handlers.ShowcaseHandler.SearchShowcases)
	api.Get("/feeds/trending", handlers.ShowcaseHandler.GetTrendingFeeds) // Trending Feeds
	api.Get("/showcases/:slug", handlers.ShowcaseHandler.GetShowcaseBySlug)
	api.Get("/showcases/user/:id", handlers.ShowcaseHandler.GetShowcasesByUser)
	api.Get("/showcases/:id/comments", handlers.ShowcaseHandler.GetShowcaseComments)

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

	protected.Post("/showcases", handlers.ShowcaseHandler.CreateShowcase)
	protected.Get(showcaseIDRoute+"/collaborators", handlers.ShowcaseHandler.GetCollaborators)
	protected.Post(showcaseIDRoute+"/collaborators", handlers.ShowcaseHandler.InviteCollaborators)
	protected.Patch(showcaseIDRoute, handlers.ShowcaseHandler.UpdateShowcase)
	protected.Delete(showcaseIDRoute+"/collaborators/:userId", handlers.ShowcaseHandler.RemoveCollaborator)
	protected.Delete(showcaseIDRoute, handlers.ShowcaseHandler.DeleteShowcase)
	protected.Post("/showcases/:id/like", handlers.ShowcaseHandler.ToggleLike)
	protected.Post("/showcases/:id/bookmark", handlers.ShowcaseHandler.ToggleBookmark)
	protected.Post("/showcases/:id/comments", handlers.ShowcaseHandler.CreateComment)
	protected.Post("/comments/:id/reply", handlers.ShowcaseHandler.ReplyComment)
	protected.Patch("/comments/:id", handlers.ShowcaseHandler.UpdateComment)
	protected.Delete("/comments/:id", handlers.ShowcaseHandler.DeleteComment)
	protected.Post("/comments/:id/like", handlers.ShowcaseHandler.ToggleCommentLike)

	protected.Delete("/collaborations/invitations/:id", handlers.ShowcaseHandler.DeleteInvitation)
	protected.Get("/collaborations/invitations", handlers.ShowcaseHandler.GetPendingInvitations)
	protected.Patch("/collaborations/:id/response", handlers.ShowcaseHandler.RespondInvitation)
	return app
}
