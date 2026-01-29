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

	Title       string         `gorm:"type:varchar(255);not null" json:"title"`
	Slug        string         `gorm:"type:varchar(300);not null;unique;index" json:"slug"`
	Content     string         `gorm:"type:text" json:"content"`
	Tags        pq.StringArray `gorm:"type:text[]" json:"tags"`
	IsEdited    bool           `gorm:"default:false" json:"is_edited"`

	// Counters
	ViewsCount    int `gorm:"default:0" json:"views_count"`
	LikesCount    int `gorm:"default:0" json:"likes_count"`
	CommentsCount int `gorm:"default:0" json:"comments_count"`
	SharesCount   int `gorm:"default:0" json:"shares_count"`

	// Relasi
	CategoryID uuid.UUID      `gorm:"type:uuid;not null" json:"category_id"`
	Category   *Category      `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"category,omitempty"` // Changed to RESTRICT
	Media      []ShowcaseMedia `gorm:"foreignKey:ShowcaseID;constraint:OnDelete:CASCADE" json:"media,omitempty"`
	Comments   []Comment      `gorm:"foreignKey:ShowcaseID;constraint:OnDelete:CASCADE" json:"comments,omitempty"` // Added Cascade
	Likes      []ShowcaseLike  `gorm:"foreignKey:ShowcaseID;constraint:OnDelete:CASCADE" json:"likes,omitempty"` // Added Cascade
	Bookmarks  []Bookmark     `gorm:"foreignKey:ShowcaseID;constraint:OnDelete:CASCADE" json:"bookmarks,omitempty"` // Added Cascade as well
	Collaborators []Collaborator `gorm:"foreignKey:ShowcaseID;constraint:OnDelete:CASCADE" json:"-"`

	// Enriched Data
	// Enriched Data
	EnrichedCollaborators []EnrichedCollaborator `gorm:"-" json:"collaborators,omitempty"`
}

type EnrichedCollaborator struct {
	ID     uuid.UUID `json:"id"`
	Role   string    `json:"role"`
	Status string    `json:"status"`
	User   *User     `json:"user"`
}

// User struct (Ghost object from User Service)
type User struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Username  string    `json:"username"`
	AvatarURL string    `json:"avatar_url"`
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
	Content     string `form:"content" validate:"required,min=10"`
	CategoryID  string `form:"category_id" validate:"required,uuid"`
	Tags        string `form:"tags"` // Comma separated: "design,ui,ux"
}

type UpdateShowcaseRequest struct {
	Title      *string  `json:"title"`
	Content    *string  `json:"content"`
	CategoryID *string  `json:"category_id"`
	Tags       []string `json:"tags"`
}
