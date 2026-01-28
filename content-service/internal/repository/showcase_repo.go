package repository

import (
	"github.com/septianpadli/talas/content-service/internal/entity"
	"gorm.io/gorm"
)

type ShowcaseRepository interface {
	Create(showcase *entity.Showcase) error
	// Add other methods (FindByID, etc)
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
