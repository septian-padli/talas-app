package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/septianpadli/talas/content-service/internal/entity"
	"github.com/septianpadli/talas/content-service/internal/repository"
)

type CategoryUsecase interface {
	ListCategories(ctx context.Context, limit int, cursor uuid.UUID) ([]entity.Category, *uuid.UUID, error)
	GetCategoryDetail(ctx context.Context, id uuid.UUID, slug string, withShowcase bool, showcaseCursor uuid.UUID, showcaseLimit int) (*entity.Category, []entity.Showcase, *uuid.UUID, error)
}

type categoryUsecase struct {
	repo repository.CategoryRepository
}

func NewCategoryUsecase(repo repository.CategoryRepository) CategoryUsecase {
	return &categoryUsecase{repo: repo}
}

func (u *categoryUsecase) ListCategories(ctx context.Context, limit int, cursor uuid.UUID) ([]entity.Category, *uuid.UUID, error) {
	return u.repo.GetCategories(limit, cursor)
}

func (u *categoryUsecase) GetCategoryDetail(ctx context.Context, id uuid.UUID, slug string, withShowcase bool, showcaseCursor uuid.UUID, showcaseLimit int) (*entity.Category, []entity.Showcase, *uuid.UUID, error) {
	category, err := u.repo.GetCategoryByIDOrSlug(id, slug)
	if err != nil {
		return nil, nil, nil, err
	}
	var showcases []entity.Showcase
	var nextCursor *uuid.UUID
	if withShowcase {
		showcases, nextCursor, err = u.repo.GetShowcasesByCategory(category.ID, showcaseCursor, showcaseLimit)
		if err != nil {
			return category, nil, nil, err
		}
	}
	return category, showcases, nextCursor, nil
}
