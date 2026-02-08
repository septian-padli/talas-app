package usecase

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"unicode"

	"github.com/google/uuid"
	"github.com/septianpadli/talas/content-service/internal/entity"
	"github.com/septianpadli/talas/content-service/internal/repository"
)

type CategoryUsecase interface {
	ListCategories(ctx context.Context, limit int, cursor uuid.UUID) ([]entity.Category, *uuid.UUID, error)
	GetCategoryDetail(ctx context.Context, id uuid.UUID, slug string, withShowcase bool, showcaseCursor uuid.UUID, showcaseLimit int) (*entity.Category, []entity.Showcase, *uuid.UUID, error)

	CreateCategory(ctx context.Context, name string) (*entity.Category, error)
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

// slugify utility
func slugify(s string) string {
	// Lowercase
	s = strings.ToLower(s)
	// Replace non-letter/digit with dash
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevDash = false
		} else if !prevDash {
			b.WriteRune('-')
			prevDash = true
		}
	}
	slug := b.String()
	// Remove leading/trailing dash
	slug = strings.Trim(slug, "-")
	// Remove duplicate dashes
	re := regexp.MustCompile(`-+`)
	slug = re.ReplaceAllString(slug, "-")
	return slug
}

func (u *categoryUsecase) CreateCategory(ctx context.Context, name string) (*entity.Category, error) {
	slug := slugify(name)
	// Cek slug sudah ada
	if existing, _ := u.repo.GetCategoryBySlug(slug); existing != nil {
		return nil, errors.New("category already exists")
	}
	category := &entity.Category{
		Base: entity.Base{ID: uuid.New()},
		Name: name,
		Slug: slug,
	}
	if err := u.repo.CreateCategory(category); err != nil {
		return nil, err
	}
	return category, nil
}
