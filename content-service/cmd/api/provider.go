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
	ShowcaseHandler *handler.ShowcaseHandler
	InternalHandler *handler.InternalHandler
	CategoryHandler *handler.CategoryHandler
	CommentHandler  *handler.CommentHandler
	CollabHandler   *handler.CollabHandler
}

func NewHandlerGroup(
	showcaseHandler *handler.ShowcaseHandler,
	internalHandler *handler.InternalHandler,
	categoryHandler *handler.CategoryHandler,
	commentHandler *handler.CommentHandler,
	collabHandler *handler.CollabHandler,
) *HandlerGroup {
	return &HandlerGroup{
		ShowcaseHandler: showcaseHandler,
		InternalHandler: internalHandler,
		CategoryHandler: categoryHandler,
		CommentHandler:  commentHandler,
		CollabHandler:   collabHandler,
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
	internal.Get(showcaseIDRoute, handlers.InternalHandler.GetShowcaseInternal)
	internal.Post("/users/bulk", handlers.InternalHandler.GetUsersBulk)

	// Category Routes (All Protected)
	api.Post("/categories", authMiddleware.Protect, handlers.CategoryHandler.CreateCategory)
	api.Get("/categories", authMiddleware.Protect, handlers.CategoryHandler.GetCategories)
	api.Get("/categories/search", authMiddleware.Protect, handlers.CategoryHandler.SearchCategories)
	api.Get("/categories/:id", authMiddleware.Protect, handlers.CategoryHandler.GetCategoryDetail)

	// Showcase & Other Routes
	api.Get("/showcases/me", authMiddleware.Protect, handlers.ShowcaseHandler.GetMyShowcases)
	api.Get("/showcases/user/:id", authMiddleware.Protect, handlers.ShowcaseHandler.GetShowcasesByUser)
	api.Get("/search", authMiddleware.Protect, handlers.ShowcaseHandler.SearchShowcases)
	api.Get("/feeds/trending", authMiddleware.Protect, handlers.ShowcaseHandler.GetTrendingFeeds)

	// public
	api.Get("/showcases/:slug", handlers.ShowcaseHandler.GetShowcaseBySlug)
	api.Get("/showcases/:id/comments", handlers.CommentHandler.GetShowcaseComments)

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
	protected.Get(showcaseIDRoute+"/collaborators", handlers.CollabHandler.GetCollaborators)
	protected.Post(showcaseIDRoute+"/collaborators", handlers.CollabHandler.InviteCollaborators)
	protected.Patch(showcaseIDRoute, handlers.ShowcaseHandler.UpdateShowcase)
	protected.Delete(showcaseIDRoute+"/collaborators/:userId", handlers.CollabHandler.RemoveCollaborator)
	protected.Delete(showcaseIDRoute, handlers.ShowcaseHandler.DeleteShowcase)
	protected.Post("/showcases/:id/like", handlers.ShowcaseHandler.ToggleLike)
	protected.Post("/showcases/:id/bookmark", handlers.ShowcaseHandler.ToggleBookmark)
	protected.Post("/showcases/:id/comments", handlers.CommentHandler.CreateComment)
	protected.Post("/comments/:id/reply", handlers.CommentHandler.ReplyComment)
	protected.Patch("/comments/:id", handlers.CommentHandler.UpdateComment)
	protected.Delete("/comments/:id", handlers.CommentHandler.DeleteComment)
	protected.Post("/comments/:id/like", handlers.CommentHandler.ToggleCommentLike)

	protected.Delete("/collaborations/invitations/:id", handlers.CollabHandler.DeleteInvitation)
	protected.Get("/collaborations/invitations", handlers.CollabHandler.GetPendingInvitations)
	protected.Patch("/collaborations/:id/response", handlers.CollabHandler.RespondInvitation)
	return app
}
