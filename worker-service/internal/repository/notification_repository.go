package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/septian/worker-service/internal/domain"
	"gorm.io/gorm"
)

type NotificationRepository interface {
	Create(ctx context.Context, notification *domain.Notification) error
}

type notificationRepository struct {
	db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) NotificationRepository {
	return &notificationRepository{
		db: db,
	}
}

// Create inserts a notification with idempotency based on event_id
func (r *notificationRepository) Create(ctx context.Context, notification *domain.Notification) error {
	err := r.db.WithContext(ctx).Create(notification).Error

	if err != nil {
		// Check if it's a unique constraint violation on event_id
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			// PostgreSQL error code 23505 = unique_violation
			if pgErr.Code == "23505" {
				// Constraint name may vary depending on migration tooling.
				// Accept any unique_violation that mentions event_id as idempotent.
				if strings.Contains(strings.ToLower(pgErr.Message), "event_id") || strings.Contains(strings.ToLower(pgErr.ConstraintName), "event_id") {
					return nil
				}
			}
		}

		return fmt.Errorf("failed to create notification: %w", err)
	}

	return nil
}
