package repository

import (
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/septianpadli/talas/content-service/internal/entity"
	"gorm.io/gorm"
)

type ShowcaseRepository interface {
	Create(showcase *entity.Showcase) error
	GetBySlug(slug string) (*entity.Showcase, error)

	GetByUserID(userID uuid.UUID, limit int, cursor string) ([]entity.Showcase, *PaginationMeta, error)
}

type paginationCursor struct {
	CreatedAt time.Time `json:"created_at"`
	ID        uuid.UUID `json:"id"`
}

type PaginationMeta struct {
	NextCursor string `json:"next_cursor"`
	HasNext    bool   `json:"has_next"`
}

type showcaseRepository struct {
	db *gorm.DB
}

func NewShowcaseRepository(db *gorm.DB) ShowcaseRepository {
	return &showcaseRepository{db: db}
}

func (r *showcaseRepository) Create(showcase *entity.Showcase) error {
	return r.db.Create(showcase).Error
}

func (r *showcaseRepository) GetBySlug(slug string) (*entity.Showcase, error) {
	var showcase entity.Showcase
	err := r.db.Preload("Category").
		Preload("Media", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC")
		}).
		Preload("Collaborators", "status = ?", "ACCEPTED").
		Where("slug = ?", slug).
		First(&showcase).Error
	if err != nil {
		return nil, err
	}
	return &showcase, nil
}

func (r *showcaseRepository) GetByUserID(userID uuid.UUID, limit int, cursor string) ([]entity.Showcase, *PaginationMeta, error) {
	var showcases []entity.Showcase
	// Query with Join on Collaborators
	query := r.db.Preload("Category").
		Preload("Media", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC")
		}).
		Preload("Collaborators").
		Joins("JOIN collaborators ON collaborators.showcase_id = showcases.id").
		Where("collaborators.user_id = ?", userID).
		Order("showcases.created_at DESC, showcases.id DESC").
		Limit(limit + 1)

	// Apply Cursor Condition

	// Apply Cursor Condition
	if cursor != "" {
		decodedBytes, err := base64.StdEncoding.DecodeString(cursor)
		if err == nil {
			var cursorObj paginationCursor
			if err := json.Unmarshal(decodedBytes, &cursorObj); err == nil {
				query = query.Where("(created_at < ?) OR (created_at = ? AND id < ?)", cursorObj.CreatedAt, cursorObj.CreatedAt, cursorObj.ID)
			}
		}
	}

	if err := query.Find(&showcases).Error; err != nil {
		return nil, nil, err
	}

	// Calculate Pagination format
	meta := &PaginationMeta{
		HasNext:    false,
		NextCursor: "",
	}

	if len(showcases) > limit {
		meta.HasNext = true
		showcases = showcases[:limit] // Remove last item (it was check for next page)
		lastItem := showcases[len(showcases)-1]

		newCursor := paginationCursor{
			CreatedAt: lastItem.CreatedAt,
			ID:        lastItem.ID,
		}
		cursorJSON, _ := json.Marshal(newCursor)
		meta.NextCursor = base64.StdEncoding.EncodeToString(cursorJSON)
	}

	return showcases, meta, nil
}
