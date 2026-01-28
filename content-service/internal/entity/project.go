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
	Projects []Project `gorm:"foreignKey:CategoryID" json:"projects,omitempty"`
}

type Project struct {
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
	Media      []ProjectMedia `gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE" json:"media,omitempty"`
	Comments   []Comment      `gorm:"foreignKey:ProjectID" json:"comments,omitempty"`
	Likes      []ProjectLike  `gorm:"foreignKey:ProjectID" json:"likes,omitempty"`
}

type ProjectMedia struct {
	Base
	ProjectID uuid.UUID `gorm:"type:uuid;not null;index" json:"project_id"`
	URL       string    `gorm:"type:text;not null" json:"url"`
	Type      string    `gorm:"type:varchar(20);default:'IMAGE'" json:"type"` // IMAGE, VIDEO
	Position  int       `gorm:"default:0" json:"position"`
}
