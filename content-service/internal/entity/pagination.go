package entity

import (
	"time"

	"github.com/google/uuid"
)

type PaginationCursor struct {
	CreatedAt time.Time `json:"created_at"`
	ID        uuid.UUID `json:"id"`
}

type PaginationMeta struct {
	NextCursor string `json:"next_cursor"`
	HasNext    bool   `json:"has_next"`
}
