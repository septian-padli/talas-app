package repository

import (
	"github.com/google/uuid"
	"github.com/septianpadli/talas/content-service/internal/entity"
	"gorm.io/gorm"
)

type CategoryRepository interface {
	GetCategories(limit int, cursor uuid.UUID) ([]entity.Category, *uuid.UUID, error)
	GetCategoryByIDOrSlug(id uuid.UUID, slug string) (*entity.Category, error)
	GetCategoryIDsBySlugs(slugs []string) ([]uuid.UUID, error)
	CreateCategory(category *entity.Category) error
	GetCategoryBySlug(slug string) (*entity.Category, error)
	SearchCategories(keyword string, limit int, cursor uuid.UUID) ([]entity.Category, *uuid.UUID, error)
}
type categoryRepository struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) CategoryRepository {
	return &categoryRepository{db: db}
}

// Infinite scroll: cursor = last UUID, ambil data setelah itu
func (r *categoryRepository) GetCategories(limit int, cursor uuid.UUID) ([]entity.Category, *uuid.UUID, error) {
	var categories []entity.Category
	query := r.db.Order("id ASC").Limit(limit)
	if cursor != uuid.Nil {
		query = query.Where("id > ?", cursor)
	}
	if err := query.Find(&categories).Error; err != nil {
		return nil, nil, err
	}
	var nextCursor *uuid.UUID
	if len(categories) == limit {
		next := categories[len(categories)-1].ID
		nextCursor = &next
	}
	return categories, nextCursor, nil
}

func (r *categoryRepository) SearchCategories(keyword string, limit int, cursor uuid.UUID) ([]entity.Category, *uuid.UUID, error) {
	var categories []entity.Category
	pattern := "%" + keyword + "%"
	// orderClause := fmt.Sprintf("POSITION(LOWER('%s') IN LOWER(name)) ASC, LENGTH(name) ASC", keyword)
	query := r.db.Where("LOWER(name) LIKE LOWER(?)", pattern).Order("id ASC").Limit(limit)
	if cursor != uuid.Nil {
		query = query.Where("id > ?", cursor)
	}
	if err := query.Find(&categories).Error; err != nil {
		return nil, nil, err
	}
	var nextCursor *uuid.UUID
	if len(categories) == limit {
		next := categories[len(categories)-1].ID
		nextCursor = &next
	}
	return categories, nextCursor, nil
}

func (r *categoryRepository) GetCategoryByIDOrSlug(id uuid.UUID, slug string) (*entity.Category, error) {
	var category entity.Category
	query := r.db
	if id != uuid.Nil {
		query = query.Where("id = ?", id)
	} else if slug != "" {
		query = query.Where("slug = ?", slug)
	} else {
		return nil, gorm.ErrRecordNotFound
	}
	if err := query.First(&category).Error; err != nil {
		return nil, err
	}
	return &category, nil
}

func (r *categoryRepository) GetCategoryBySlug(slug string) (*entity.Category, error) {
	var category entity.Category
	if err := r.db.Where("slug = ?", slug).First(&category).Error; err != nil {
		return nil, err
	}
	return &category, nil
}

func (r *categoryRepository) CreateCategory(category *entity.Category) error {
	return r.db.Create(category).Error
}

func (r *categoryRepository) GetCategoryIDsBySlugs(slugs []string) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := r.db.Model(&entity.Category{}).Where("slug IN ?", slugs).Pluck("id", &ids).Error
	return ids, err
}
