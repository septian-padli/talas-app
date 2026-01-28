package entity

import (
	"time"

	"github.com/google/uuid"
)

const (
	CollaborationStatusPending  = "PENDING"
	CollaborationStatusAccepted = "ACCEPTED"
	CollaborationStatusRejected = "REJECTED"
)

type Collaborator struct {
	Base
	ProjectID uuid.UUID `gorm:"type:uuid;not null;index" json:"project_id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"` // Ghost FK

	Role      string    `gorm:"type:varchar(50);default:'EDITOR'" json:"role"` // OWNER, EDITOR, VIEWER
	Status    string    `gorm:"type:varchar(20);default:'PENDING'" json:"status"`
	ExpiredAt time.Time `json:"expired_at"` // Invitation expiry
}
