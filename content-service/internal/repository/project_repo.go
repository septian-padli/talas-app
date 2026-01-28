package repository

import (
	"github.com/septianpadli/talas/content-service/internal/entity"
	"gorm.io/gorm"
)

type ProjectRepository interface {
	Create(project *entity.Project) error
	// Add other methods (FindByID, etc)
}

type projectRepository struct {
	db *gorm.DB
}

func NewProjectRepository(db *gorm.DB) ProjectRepository {
	return &projectRepository{db: db}
}

func (r *projectRepository) Create(project *entity.Project) error {
	return r.db.Create(project).Error
}
