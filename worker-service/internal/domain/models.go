package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Collaborator represents a user in a showcase
type Collaborator struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	FullName  string `json:"full_name"`
	AvatarURL string `json:"avatar_url"`
}

// UserSimple represents basic user info
type UserSimple struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	FullName  string `json:"full_name"`
	AvatarURL string `json:"avatar_url"`
}

// EventEnvelope is the wrapper format from content-service events
type EventEnvelope struct {
	EventID   string          `json:"event_id"`
	EventType string          `json:"event_type"`
	Timestamp time.Time       `json:"timestamp"`
	Data      json.RawMessage `json:"data"` // Use RawMessage for delayed unmarshal
}

// ShowcaseEventData represents the data field in showcase events
type ShowcaseEventData struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	Slug      string    `json:"slug"`
	UserID    uuid.UUID `json:"user_id"`
	AvatarURL string    `json:"avatar_url"`
}

// CollaboratorEventData represents the payload for collaborator.responded
type CollaboratorEventData struct {
	InvitationID       string `json:"invitation_id"`
	ShowcaseID         string `json:"showcase_id"`
	ShowcaseTitle      string `json:"showcase_title"`
	ResponseStatus     string `json:"response_status"`
	ResponderID        string `json:"responder_id"`
	ResponderUsername  string `json:"responder_username"`
	ResponderFullName  string `json:"responder_full_name"`
	ResponderAvatarURL string `json:"responder_avatar_url"`
	TargetUserID       string `json:"target_user_id"`
}

// CollaboratorRemoveData represents the payload for collaborator.removed
type CollaboratorRemoveData struct {
	ShowcaseID      string `json:"showcase_id"`
	ActorID         string `json:"actor_id"`
	TargetUserID    string `json:"target_user_id"`
	IsSelfRemoval   bool   `json:"is_self_removal"`
	ShowcaseOwnerID string `json:"showcase_owner_id"`
}

// Showcase represents the showcase data to be indexed in Elasticsearch
type Showcase struct {
	ID            uuid.UUID      `json:"id"`
	Title         string         `json:"title"`
	Slug          string         `json:"slug"`
	Tags          []string       `json:"tags"`
	Description   string         `json:"description,omitempty"`
	Content       string         `json:"content,omitempty"`
	Thumbnail     string         `json:"thumbnail,omitempty"`
	CategoryID    uuid.UUID      `json:"category_id,omitempty"`
	Owner         UserSimple     `json:"owner,omitempty"`
	Collaborators []Collaborator `json:"collaborators,omitempty"`
	LikeCount     int            `json:"like_count"`
	ViewCount     int            `json:"view_count"`
	CreatedAt     time.Time      `json:"created_at,omitempty"`
	UpdatedAt     time.Time      `json:"updated_at,omitempty"`
}

// Notification represents a notification to be stored in the database
//
//	type Notification struct {
//		ID        uuid.UUID              `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
//		EventID   string                 `gorm:"uniqueIndex;not null"`
//		UserID    uuid.UUID              `gorm:"type:uuid;not null"`
//		Type      string                 `gorm:"type:varchar(50);not null"`
//		Title     string                 `gorm:"type:varchar(255);not null"`
//		Message   string                 `gorm:"type:text;not null"`
//		Data      map[string]interface{} `gorm:"type:jsonb"`
//		Read      bool                   `gorm:"default:false"`
//		CreatedAt time.Time              `gorm:"autoCreateTime"`
//	}
type Notification struct {
	ID       string  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	EventID  string  `gorm:"type:uuid;unique" json:"event_id"` // Pastikan field ini ada
	Type     string  `json:"type"`
	IsRead   bool    `json:"is_read" gorm:"column:is_read;default:false"`
	UserID   string  `gorm:"type:uuid" json:"user_id"`
	SenderID *string `gorm:"type:uuid" json:"sender_id"`

	// GANTI Title & Message dengan INI:
	Data json.RawMessage `gorm:"type:jsonb" json:"data"`

	// Prisma schema uses `createdAt` (camelCase) column name.
	CreatedAt time.Time `json:"created_at" gorm:"column:createdAt;autoCreateTime"`

	// `updated_at` was removed in Prisma migrations; ignore UpdatedAt in GORM to avoid inserting missing column.
	// If you later add updatedAt column to DB, re-add this field.
	// UpdatedAt time.Time `json:"updated_at" gorm:"column:updatedAt;autoUpdateTime"`
}
