package domain

import (
	"time"

	"github.com/google/uuid"
)

// Collaborator represents a user in a showcase
type Collaborator struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

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

// CollaboratorEventData represents the payload for collaborator.responded
type CollaboratorEventData struct {
	InvitationID      string `json:"invitation_id"`
	ShowcaseID        string `json:"showcase_id"`
	ShowcaseTitle     string `json:"showcase_title"`
	ResponseStatus    string `json:"response_status"`
	ResponderID       string `json:"responder_id"`
	ResponderUsername string `json:"responder_username"`
	TargetUserID      string `json:"target_user_id"`
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
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Slug        string    `json:"slug"`
	Tags        []string  `json:"tags"`
	Description string    `json:"description,omitempty"`
	Content     string    `json:"content,omitempty"`
	Thumbnail   string    `json:"thumbnail,omitempty"`
	CategoryID  uuid.UUID `json:"category_id,omitempty"`
	OwnerID     uuid.UUID `json:"owner_id,omitempty"`
	OwnerName     string         `json:"owner_name,omitempty"`
	Collaborators []Collaborator `json:"collaborators,omitempty"`
	LikeCount     int            `json:"like_count"`
	ViewCount     int            `json:"view_count"`
	CreatedAt     time.Time      `json:"created_at,omitempty"`
	UpdatedAt     time.Time      `json:"updated_at,omitempty"`
}
