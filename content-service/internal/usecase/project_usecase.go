package usecase

import (
	"github.com/septianpadli/talas/content-service/internal/entity"
	"github.com/septianpadli/talas/content-service/internal/repository"
)

type ProjectUsecase interface {
	CreateProject(input *entity.Project) error
}

type projectUsecase struct {
	repo repository.ProjectRepository
}

func NewProjectUsecase(repo repository.ProjectRepository) ProjectUsecase {
	return &projectUsecase{repo: repo}
}

func (u *projectUsecase) CreateProject(input *entity.Project) error {
	// Add Validation Logic Here
	return u.repo.Create(input)
}
