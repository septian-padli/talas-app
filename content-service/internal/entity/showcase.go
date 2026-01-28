package entity

import (
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// STATUS ENUM
const (
	StatusDraft     = "DRAFT"
	StatusPublished = "PUBLISHED"
	StatusArchived  = "ARCHIVED"
)

type Category struct {
	Base
	Name string `gorm:"type:varchar(100);not null;unique" json:"name"`
	Slug string `gorm:"type:varchar(150);not null;unique;index" json:"slug"`

	// Relasi
	Showcases []Showcase `gorm:"foreignKey:CategoryID" json:"showcases,omitempty"`
}

type Showcase struct {
	Base
	UserID uuid.UUID `gorm:"type:uuid;index;not null" json:"user_id"` // Ghost FK ke User Service

	Title       string         `gorm:"type:varchar(255);not null" json:"title"`
	Slug        string         `gorm:"type:varchar(300);not null;unique;index" json:"slug"`
	Description string         `gorm:"type:text" json:"description"`
	Status      string         `gorm:"type:varchar(20);default:'PUBLISHED';index" json:"status"`
	Tags        pq.StringArray `gorm:"type:text[]" json:"tags"` // Array of strings (Postgres Native)

	// Counters (Denormalization for Performance)
	ViewsCount    int `gorm:"default:0" json:"views_count"`
	LikesCount    int `gorm:"default:0" json:"likes_count"`
	CommentsCount int `gorm:"default:0" json:"comments_count"`
	SharesCount   int `gorm:"default:0" json:"shares_count"` // Optional

	// Relasi
	CategoryID uuid.UUID      `gorm:"type:uuid;not null" json:"category_id"`
	Category   *Category      `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"category,omitempty"`
	Media      []ShowcaseMedia `gorm:"foreignKey:ShowcaseID;constraint:OnDelete:CASCADE" json:"media,omitempty"`
	Comments   []Comment      `gorm:"foreignKey:ShowcaseID" json:"comments,omitempty"`
	Likes      []ShowcaseLike  `gorm:"foreignKey:ShowcaseID" json:"likes,omitempty"`
}

type ShowcaseMedia struct {
	Base
	ShowcaseID uuid.UUID `gorm:"type:uuid;not null;index" json:"showcase_id"`
	URL       string    `gorm:"type:text;not null" json:"url"`
	Type      string    `gorm:"type:varchar(20);default:'IMAGE'" json:"type"` // IMAGE, VIDEO
	Position  int       `gorm:"default:0" json:"position"`
}

// DTOs
type CreateShowcaseRequest struct {
	Title       string `form:"title" validate:"required,min=5,max=100"`
	Description string `form:"description" validate:"required,min=10"`
	CategoryID  string `form:"category_id" validate:"required,uuid"`
	Tags        string `form:"tags"` // Comma separated: "design,ui,ux"
}
