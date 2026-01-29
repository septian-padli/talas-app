package entity

import (
	"time"

	"github.com/google/uuid"
)

const (
	CollaborationStatusPending  = "PENDING"
	CollaborationStatusAccepted = "ACCEPTED"
	CollaborationStatusRejected = "REJECTED"

	CollaborationRoleOwner        = "OWNER"
	CollaborationRoleCollaborator = "COLLABORATOR"
)

type Collaborator struct {
	Base
	ShowcaseID uuid.UUID `gorm:"type:uuid;not null;index" json:"showcase_id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"` // Ghost FK

	Role      string    `gorm:"type:varchar(50);default:'EDITOR'" json:"role"` // OWNER, EDITOR, VIEWER
	Status    string    `gorm:"type:varchar(20);default:'PENDING'" json:"status"`
	ExpiredAt time.Time `json:"expired_at"` // Invitation expiry

	// Relations
	Showcase *Showcase `gorm:"foreignKey:ShowcaseID;constraint:OnDelete:CASCADE" json:"showcase,omitempty"`
	User     *User     `gorm:"foreignKey:UserID;references:ID;constraint:-" json:"user,omitempty"`
}
