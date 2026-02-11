package repository

import (
	"encoding/base64"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/septianpadli/talas/content-service/internal/entity"
	"gorm.io/gorm"
)

type ShowcaseRepository interface {
	Create(showcase *entity.Showcase) error
	Update(showcase *entity.Showcase) error
	GetBySlug(slug string) (*entity.Showcase, error)
	GetByID(id uuid.UUID) (*entity.Showcase, error)

	GetByUserID(userID uuid.UUID, limit int, cursor string) ([]entity.Showcase, *entity.PaginationMeta, error)
	ToggleLike(userID uuid.UUID, showcaseID uuid.UUID) (bool, error)
	ToggleBookmark(userID uuid.UUID, showcaseID uuid.UUID) (bool, error)
	DeleteShowcase(id uuid.UUID) error
	GetCategoryIDsBySlugs(slugs []string) ([]uuid.UUID, error)
	IncrementViewCount(id uuid.UUID) error
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

func (r *showcaseRepository) Update(showcase *entity.Showcase) error {
	return r.db.Save(showcase).Error
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

func (r *showcaseRepository) GetByID(id uuid.UUID) (*entity.Showcase, error) {
	var showcase entity.Showcase
	err := r.db.Preload("Category").
		Preload("Media", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC")
		}).
		Preload("Collaborators", "status = ?", "ACCEPTED").
		Where("id = ?", id).
		First(&showcase).Error
	if err != nil {
		return nil, err
	}
	return &showcase, nil
}

func (r *showcaseRepository) GetByUserID(userID uuid.UUID, limit int, cursor string) ([]entity.Showcase, *entity.PaginationMeta, error) {
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
			var cursorObj entity.PaginationCursor
			if err := json.Unmarshal(decodedBytes, &cursorObj); err == nil {
				query = query.Where("(created_at < ?) OR (created_at = ? AND id < ?)", cursorObj.CreatedAt, cursorObj.CreatedAt, cursorObj.ID)
			}
		}
	}

	if err := query.Find(&showcases).Error; err != nil {
		return nil, nil, err
	}

	// Calculate Pagination format
	meta := &entity.PaginationMeta{
		HasNext:    false,
		NextCursor: "",
	}

	if len(showcases) > limit {
		meta.HasNext = true
		showcases = showcases[:limit] // Remove last item (it was check for next page)
		lastItem := showcases[len(showcases)-1]

		newCursor := entity.PaginationCursor{
			CreatedAt: lastItem.CreatedAt,
			ID:        lastItem.ID,
		}
		cursorJSON, _ := json.Marshal(newCursor)
		meta.NextCursor = base64.StdEncoding.EncodeToString(cursorJSON)
	}

	return showcases, meta, nil
}

func (r *showcaseRepository) ToggleLike(userID uuid.UUID, showcaseID uuid.UUID) (bool, error) {
	var like entity.ShowcaseLike
	var isLiked bool

	err := r.db.Transaction(func(tx *gorm.DB) error {
		// Use Limit(1).Find() instead of First() to avoid "record not found" log error
		result := tx.Where("user_id = ? AND showcase_id = ?", userID, showcaseID).Limit(1).Find(&like)

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected > 0 {
			// Found -> Delete (Unlike)
			if err := tx.Delete(&like).Error; err != nil {
				return err
			}
			// Decrement Count
			if err := tx.Model(&entity.Showcase{}).Where("id = ?", showcaseID).UpdateColumn("likes_count", gorm.Expr("likes_count - ?", 1)).Error; err != nil {
				return err
			}
			isLiked = false
		} else {
			// Not Found (RowsAffected == 0) -> Create (Like)
			newLike := entity.ShowcaseLike{
				UserID:     userID,
				ShowcaseID: showcaseID,
			}
			if err := tx.Create(&newLike).Error; err != nil {
				return err
			}
			// Increment Count
			if err := tx.Model(&entity.Showcase{}).Where("id = ?", showcaseID).UpdateColumn("likes_count", gorm.Expr("likes_count + ?", 1)).Error; err != nil {
				return err
			}
			isLiked = true
		}
		return nil
	})

	return isLiked, err
}

func (r *showcaseRepository) ToggleBookmark(userID uuid.UUID, showcaseID uuid.UUID) (bool, error) {
	var bookmark entity.Bookmark
	var isBookmarked bool

	err := r.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("user_id = ? AND showcase_id = ?", userID, showcaseID).Limit(1).Find(&bookmark)

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected > 0 {
			// Found -> Delete (Unbookmark)
			// Hard delete implicit (no DeletedAt field)
			if err := tx.Delete(&bookmark).Error; err != nil {
				return err
			}
			isBookmarked = false
		} else {
			// Not Found -> Create (Bookmark)
			newBookmark := entity.Bookmark{
				UserID:     userID,
				ShowcaseID: showcaseID,
			}
			if err := tx.Create(&newBookmark).Error; err != nil {
				return err
			}
			isBookmarked = true
		}
		return nil
	})

	return isBookmarked, err
}

func (r *showcaseRepository) DeleteShowcase(id uuid.UUID) error {
	return r.db.Delete(&entity.Showcase{}, id).Error
}

func (r *showcaseRepository) GetCategoryIDsBySlugs(slugs []string) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := r.db.Model(&entity.Category{}).Where("slug IN ?", slugs).Pluck("id", &ids).Error
	return ids, err
}

func (r *showcaseRepository) IncrementViewCount(id uuid.UUID) error {
	return r.db.Model(&entity.Showcase{}).Where("id = ?", id).UpdateColumn("views_count", gorm.Expr("views_count + ?", 1)).Error
}
