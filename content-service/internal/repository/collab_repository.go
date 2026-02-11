package repository

import (
	"github.com/google/uuid"
	"github.com/septianpadli/talas/content-service/internal/entity"
	"github.com/septianpadli/talas/content-service/pkg/utils"
	"gorm.io/gorm"
)

type CollabRepository interface {
	DeleteCollaborator(showcaseID, userID uuid.UUID) error
	GetCollaboratorsByShowcaseID(showcaseID uuid.UUID) ([]entity.Collaborator, error)
	GetCollaboratorByID(id uuid.UUID) (*entity.Collaborator, error)
	UpdateCollaboratorRole(showcaseID, userID uuid.UUID, role string) error
	DeleteCollaboratorByID(id uuid.UUID) error
	GetPendingInvitations(userID uuid.UUID, limit int, cursor string) ([]entity.Collaborator, *entity.PaginationMeta, error)
	AddCollaborators(collaborators []entity.Collaborator) error
	UpdateCollaboratorStatus(id uuid.UUID, status string) error
	IncrementViewCount(id uuid.UUID) error
}

type collabRepository struct {
	db *gorm.DB
}

func NewCollabRepository(db *gorm.DB) CollabRepository {
	return &collabRepository{db: db}
}

func (r *collabRepository) DeleteCollaborator(showcaseID, userID uuid.UUID) error {
	// Hard Delete (as per rule: Remove access completely)
	// Or soft? Use hard delete for cleanup as Collaborator table doesn't have DeletedAt usually?
	// Checking entity definition later. Assuming Hard Delete for relation table.
	return r.db.Where("showcase_id = ? AND user_id = ?", showcaseID, userID).Delete(&entity.Collaborator{}).Error
}

func (r *collabRepository) GetCollaboratorsByShowcaseID(showcaseID uuid.UUID) ([]entity.Collaborator, error) {
	var collaborators []entity.Collaborator
	err := r.db.Where("showcase_id = ?", showcaseID).
		Order("created_at ASC").
		Find(&collaborators).Error
	return collaborators, err
}

func (r *collabRepository) GetCollaboratorByID(id uuid.UUID) (*entity.Collaborator, error) {
	var col entity.Collaborator
	// Preload Showcase? Not needed if we use GetByID separately or if logic is separate.
	// But actually, checking strict equality of owner is easier if we fetch showcase separately in usecase.
	err := r.db.Where("id = ?", id).First(&col).Error
	if err != nil {
		return nil, err
	}
	return &col, nil
}

func (r *collabRepository) UpdateCollaboratorRole(showcaseID, userID uuid.UUID, role string) error {
	return r.db.Model(&entity.Collaborator{}).
		Where("showcase_id = ? AND user_id = ?", showcaseID, userID).
		Update("role", role).Error
}

func (r *collabRepository) DeleteCollaboratorByID(id uuid.UUID) error {
	return r.db.Delete(&entity.Collaborator{}, id).Error
}

func (r *collabRepository) GetPendingInvitations(userID uuid.UUID, limit int, cursor string) ([]entity.Collaborator, *entity.PaginationMeta, error) {
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

func (r *collabRepository) AddCollaborators(collaborators []entity.Collaborator) error {
	return r.db.Create(&collaborators).Error
}

func (r *collabRepository) UpdateCollaboratorStatus(id uuid.UUID, status string) error {
	return r.db.Model(&entity.Collaborator{}).Where("id = ?", id).Update("status", status).Error
}

func (r *collabRepository) IncrementViewCount(id uuid.UUID) error {
	return r.db.Model(&entity.Showcase{}).Where("id = ?", id).UpdateColumn("views_count", gorm.Expr("views_count + ?", 1)).Error
}
