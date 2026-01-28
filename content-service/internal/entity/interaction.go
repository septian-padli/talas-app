package entity

import (
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
}

type ShowcaseLike struct {
	Base
	ShowcaseID uuid.UUID `gorm:"type:uuid;not null;index:idx_showcase_like_unique,unique" json:"showcase_id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index:idx_showcase_like_unique,unique" json:"user_id"`
}

type CommentLike struct {
	Base
	CommentID uuid.UUID `gorm:"type:uuid;not null;index:idx_comment_like_unique,unique" json:"comment_id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index:idx_comment_like_unique,unique" json:"user_id"`
}

type Bookmark struct {
	Base
	ShowcaseID uuid.UUID `gorm:"type:uuid;not null;index:idx_bookmark_unique,unique" json:"showcase_id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index:idx_bookmark_unique,unique" json:"user_id"`
}
