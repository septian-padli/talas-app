package entity

import (
	"time"

	"github.com/google/uuid"
)

type Comment struct {
	Base
	ShowcaseID uuid.UUID `gorm:"type:uuid;not null;index" json:"showcase_id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"` // Ghost FK ke User Service

	Body string `gorm:"type:text;not null" json:"body"`

	// Nested Replies (Flattened Strategy per context)
	ParentID *uuid.UUID `gorm:"type:uuid;index" json:"parent_id"` // Jika reply, simpan ID comment parent
	ReplyTo  string     `gorm:"type:varchar(100)" json:"reply_to"` // Username yang direply (snapshot)

	LikesCount int `gorm:"default:0" json:"likes_count"`

	Author *User `gorm:"-" json:"author,omitempty"`
	AuthorID *uuid.UUID `gorm:"type:uuid" json:"author_id,omitempty"` // For guest? Or just UserID.
	IsEdited bool       `gorm:"default:false" json:"is_edited"`
	Replies []Comment `gorm:"-" json:"replies,omitempty"`
}

type ShowcaseLike struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	// No DeletedAt (Hard Delete)

	ShowcaseID uuid.UUID `gorm:"type:uuid;not null;index:idx_showcase_like_unique,unique" json:"showcase_id"`
	UserID     uuid.UUID `gorm:"type:uuid;not null;index:idx_showcase_like_unique,unique" json:"user_id"`
}

type CommentLike struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	// No DeletedAt (Hard Delete)

	CommentID uuid.UUID `gorm:"type:uuid;not null;index:idx_comment_like_unique,unique" json:"comment_id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index:idx_comment_like_unique,unique" json:"user_id"`
}

type Bookmark struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	ShowcaseID uuid.UUID `gorm:"type:uuid;not null;index:idx_bookmark_unique,unique" json:"showcase_id"`
	UserID     uuid.UUID `gorm:"type:uuid;not null;index:idx_bookmark_unique,unique" json:"user_id"`
}

type CreateCommentRequest struct {
	Content  string  `json:"content" validate:"required,min=1,max=1000"`
	ParentID *string `json:"parent_id"` // Optional UUID string
}

type ReplyCommentRequest struct {
	Content string `json:"content" validate:"required,min=1,max=1000"`
}

type UpdateCommentRequest struct {
	Content string `json:"content" validate:"required,min=1,max=1000"`
}
