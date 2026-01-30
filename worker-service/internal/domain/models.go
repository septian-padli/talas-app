package domain

import (
	"time"

	"github.com/google/uuid"
)

// EventEnvelope is the wrapper format from content-service events
type EventEnvelope struct {
	EventID   string                 `json:"event_id"`
	EventType string                 `json:"event_type"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// ShowcaseEventData represents the data field in showcase events
type ShowcaseEventData struct {
	ID     uuid.UUID `json:"id"`
	Title  string    `json:"title"`
	Slug   string    `json:"slug"`
	UserID uuid.UUID `json:"user_id"`
}

// Showcase represents the showcase data to be indexed in Elasticsearch
type Showcase struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Slug        string    `json:"slug"`
	Description string    `json:"description,omitempty"`
	Content     string    `json:"content,omitempty"`
	Thumbnail   string    `json:"thumbnail,omitempty"`
	CategoryID  uuid.UUID `json:"category_id,omitempty"`
	OwnerID     uuid.UUID `json:"owner_id,omitempty"`
	OwnerName   string    `json:"owner_name,omitempty"`
	LikeCount   int       `json:"like_count"`
	ViewCount   int       `json:"view_count"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
}
