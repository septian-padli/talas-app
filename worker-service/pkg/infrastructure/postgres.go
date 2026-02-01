package infrastructure

import (
	"fmt"

	"github.com/septian/worker-service/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// NewPostgresDB creates a new PostgreSQL database connection
func NewPostgresDB(cfg *config.Config) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return db, nil
}
