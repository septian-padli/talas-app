package integration

import (
	"context"
	"fmt"
	"mime/multipart"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/septianpadli/talas/content-service/internal/config"
	"github.com/septianpadli/talas/content-service/internal/entity"
	"github.com/septianpadli/talas/content-service/internal/handler"
	"github.com/septianpadli/talas/content-service/internal/repository"
	"github.com/septianpadli/talas/content-service/internal/usecase"
	"github.com/septianpadli/talas/content-service/pkg/clients"
	"github.com/septianpadli/talas/content-service/pkg/logger"
	"github.com/septianpadli/talas/content-service/pkg/middleware"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// --- MOCKS ---

// MockMediaUploader avoids hitting Cloudinary
type MockMediaUploader struct{}

func (m *MockMediaUploader) Upload(ctx context.Context, file multipart.File, filename string, folder string) (string, error) {
	// Return dummy URL
	return "https://res.cloudinary.com/demo/image/upload/v1/dummy/" + filename, nil
}

// MockUserClient avoids hitting User Service
type MockUserClient struct{}

func (m *MockUserClient) GetUsersBulk(userIDs []uuid.UUID) (map[uuid.UUID]clients.UserDetail, error) {
	result := make(map[uuid.UUID]clients.UserDetail)
	for _, id := range userIDs {
		result[id] = clients.UserDetail{
			ID:        id,
			Name:      "Test User",
			Username:  "testuser",
			AvatarURL: "https://example.com/avatar.jpg",
		}
	}
	return result, nil
}

func (m *MockUserClient) GetUsersByUsernames(usernames []string) (map[string]clients.UserDetail, error) {
	result := make(map[string]clients.UserDetail)
	for _, username := range usernames {
		result[username] = clients.UserDetail{
			ID:        uuid.New(),
			Name:      "Test User",
			Username:  username,
			AvatarURL: "https://example.com/avatar.jpg",
		}
	}
	return result, nil
}

// MockEventPublisher is a mock implementation of rabbitmq.EventPublisher
type MockEventPublisher struct {
	Events []map[string]interface{}
}

func (m *MockEventPublisher) Publish(ctx context.Context, routingKey string, data interface{}) error {
	m.Events = append(m.Events, map[string]interface{}{
		"routingKey": routingKey,
		"data":       data,
	})
	return nil
}

func (m *MockEventPublisher) Close() error {
	return nil
}

// MockSearchRepository is a mock implementation of repository.SearchRepository
type MockSearchRepository struct{}

func (m *MockSearchRepository) SearchShowcases(ctx context.Context, query string, page int, limit int) ([]entity.Showcase, int64, error) {
	// Return empty result for now
	return []entity.Showcase{}, 0, nil
}

// --- SETUP HELPERS ---

var testDB *gorm.DB

func setupTestDB() *gorm.DB {
	if testDB != nil {
		return testDB
	}

	cfg := config.LoadConfig()
	// Override DB Name for testing
	cfg.DBName = "talas_content_test" 
	
	// Reconstruct DSN manually or use database package if we modify it.
	// For now, let's just construct it here to allow custom GORM config
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		cfg.DBHost,
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBName,
		cfg.DBPort,
		cfg.DBSSLMode,
	)

	// Connect (and AutoMigrate)
	// Silence GORM logs
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		panic("Failed to connect to test DB: " + err.Error())
	}
	
	// Auto Migrate (Copied directly to ensure test DB has tables)
	err = db.AutoMigrate(
		&entity.Category{},
		&entity.Showcase{},
		&entity.ShowcaseMedia{},
		&entity.Comment{},
		&entity.ShowcaseLike{},
		&entity.CommentLike{},
		&entity.Bookmark{},
		&entity.Collaborator{},
	)
	if err != nil {
		panic("Failed to migrate test DB: " + err.Error())
	}

	// Seeding Categories for Testing
	seedCategories(db)

	testDB = db
	return db
}

func seedCategories(db *gorm.DB) {
	categories := []entity.Category{
		{Name: "Technology", Slug: "technology"},
		{Name: "Design", Slug: "design"},
	}
	for _, cat := range categories {
		db.FirstOrCreate(&cat, entity.Category{Slug: cat.Slug})
	}
}

// setupIntegrationApp initializes the app (Backward Compatibility)
func setupIntegrationApp() (*fiber.App, *gorm.DB) {
	app, db, _ := setupIntegrationAppWithMock()
	return app, db
}

// setupIntegrationAppWithMock initializes the app with Real DB but Mocked External Services
func setupIntegrationAppWithMock() (*fiber.App, *gorm.DB, *MockEventPublisher) {
	// 1. Config & Logger
	cfg := config.LoadConfig()
	cfg.DBName = "talas_content_test" // Ensure config used by others also points to test DB
	log := logger.NewLogger()

	// 2. Database
	db := setupTestDB()

	// 3. Repository
	repo := repository.NewShowcaseRepository(db)

	// 4. Mocks
	mockUploader := &MockMediaUploader{}
	mockUserClient := &MockUserClient{}
	mockEventPublisher := &MockEventPublisher{}
	mockSearchRepo := &MockSearchRepository{}

	// 5. Usecase (Injected with Mocks)
	uc := usecase.NewShowcaseUsecase(repo, mockSearchRepo, mockUserClient, mockUploader, mockEventPublisher, cfg, log)

	// 6. Handler
	h := handler.NewShowcaseHandler(uc, log)

	// 7. Middleware
	auth := middleware.NewAuthMiddleware(cfg)

	// 8. Fiber App (Manual Wiring of NewFiberApp logic to avoid import cycle or use provider)
	// We can use the NewFiberApp from main package? No, it's in main package.
	// We should duplicate the wiring route logic OR move NewFiberApp to a shared package.
	// Since NewFiberApp is in `cmd/api`, we cannot import it easily due to `main` package constraint usually.
	// However, `cmd/api/provider.go` has package `main`. Integration tests are in `integration` package.
	// We cannot import `main` package.
	// So we must Construct the App here manually.

	app := fiber.New(fiber.Config{
		BodyLimit: 20 * 1024 * 1024,
	})

	// Register Routes manually (Copied from provider.go)
	api := app.Group("/api")
	
	// Protected Group
	protected := api.Group("/")
	protected.Use(auth.Protect)

	protected.Post("/showcases", h.CreateShowcase)
	protected.Patch("/showcases/:id", h.UpdateShowcase)
	protected.Delete("/showcases/:id", h.DeleteShowcase)
	
	protected.Post("/showcases/:id/collaborators", h.InviteCollaborators)
	protected.Delete("/showcases/:id/collaborators/:userId", h.RemoveCollaborator)
	protected.Get("/showcases/:id/collaborators", h.GetCollaborators)
	
	protected.Patch("/collaborations/:id/response", h.RespondInvitation)
	
	protected.Post("/showcases/:id/like", h.ToggleLike)
	protected.Post("/showcases/:id/bookmark", h.ToggleBookmark)
	protected.Post("/comments/:id/like", h.ToggleCommentLike)
	
	protected.Post("/showcases/:id/comments", h.CreateComment)
	
	protected.Post("/comments/:id/reply", h.ReplyComment)
	protected.Delete("/comments/:id", h.DeleteComment)
	// Add other routes as needed for tests

	return app, db, mockEventPublisher
}
