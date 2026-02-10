package repository

import (
	"encoding/base64"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/septianpadli/talas/content-service/internal/entity"
	"github.com/septianpadli/talas/content-service/pkg/utils"
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
	DeleteCollaborator(showcaseID, userID uuid.UUID) error
	GetCollaboratorsByShowcaseID(showcaseID uuid.UUID) ([]entity.Collaborator, error)
	GetCollaboratorByID(id uuid.UUID) (*entity.Collaborator, error)
	UpdateCollaboratorRole(showcaseID, userID uuid.UUID, role string) error
	DeleteCollaboratorByID(id uuid.UUID) error
	GetPendingInvitations(userID uuid.UUID, limit int, cursor string) ([]entity.Collaborator, *entity.PaginationMeta, error)
	AddCollaborators(collaborators []entity.Collaborator) error
	UpdateCollaboratorStatus(id uuid.UUID, status string) error
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

func (r *showcaseRepository) DeleteCollaborator(showcaseID, userID uuid.UUID) error {
	// Hard Delete (as per rule: Remove access completely)
	// Or soft? Use hard delete for cleanup as Collaborator table doesn't have DeletedAt usually?
	// Checking entity definition later. Assuming Hard Delete for relation table.
	return r.db.Where("showcase_id = ? AND user_id = ?", showcaseID, userID).Delete(&entity.Collaborator{}).Error
}

func (r *showcaseRepository) GetCollaboratorsByShowcaseID(showcaseID uuid.UUID) ([]entity.Collaborator, error) {
	var collaborators []entity.Collaborator
	err := r.db.Preload("User").
		Where("showcase_id = ?", showcaseID).
		Order("created_at ASC").
		Find(&collaborators).Error
	return collaborators, err
}

func (r *showcaseRepository) GetCollaboratorByID(id uuid.UUID) (*entity.Collaborator, error) {
	var col entity.Collaborator
	// Preload Showcase? Not needed if we use GetByID separately or if logic is separate.
	// But actually, checking strict equality of owner is easier if we fetch showcase separately in usecase.
	err := r.db.Where("id = ?", id).First(&col).Error
	if err != nil {
		return nil, err
	}
	return &col, nil
}

func (r *showcaseRepository) UpdateCollaboratorRole(showcaseID, userID uuid.UUID, role string) error {
	return r.db.Model(&entity.Collaborator{}).
		Where("showcase_id = ? AND user_id = ?", showcaseID, userID).
		Update("role", role).Error
}

func (r *showcaseRepository) DeleteCollaboratorByID(id uuid.UUID) error {
	return r.db.Delete(&entity.Collaborator{}, id).Error
}

func (r *showcaseRepository) GetPendingInvitations(userID uuid.UUID, limit int, cursor string) ([]entity.Collaborator, *entity.PaginationMeta, error) {
	var invitations []entity.Collaborator
	query := r.db.Preload("Showcase").
		// Preload Showcase Owner to map as Inviter later?
		// We can try loading Showcase.Collaborators
		Preload("Showcase.Collaborators").
		Where("user_id = ? AND status = ?", userID, entity.CollaborationStatusPending).
		Order("created_at DESC")

	if cursor != "" {
		cursorTime, err := utils.DecodeCursor(cursor)
		if err == nil {
			query = query.Where("created_at < ?", cursorTime)
		}
	}

	err := query.Limit(limit + 1).Find(&invitations).Error
	if err != nil {
		return nil, nil, err
	}

	meta := &entity.PaginationMeta{HasNext: false}
	if len(invitations) > limit {
		meta.HasNext = true
		meta.NextCursor = utils.EncodeCursor(invitations[limit].CreatedAt)
		invitations = invitations[:limit]
	}

	return invitations, meta, nil
}

func (r *showcaseRepository) AddCollaborators(collaborators []entity.Collaborator) error {
	return r.db.Create(&collaborators).Error
}

func (r *showcaseRepository) UpdateCollaboratorStatus(id uuid.UUID, status string) error {
	return r.db.Model(&entity.Collaborator{}).Where("id = ?", id).Update("status", status).Error
}

func (r *showcaseRepository) GetCategoryIDsBySlugs(slugs []string) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := r.db.Model(&entity.Category{}).Where("slug IN ?", slugs).Pluck("id", &ids).Error
	return ids, err
}

func (r *showcaseRepository) IncrementViewCount(id uuid.UUID) error {
	return r.db.Model(&entity.Showcase{}).Where("id = ?", id).UpdateColumn("views_count", gorm.Expr("views_count + ?", 1)).Error
}
