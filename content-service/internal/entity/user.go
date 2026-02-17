package entity

import (
	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Username  string    `json:"username"`
	JobTitle  string    `json:"jobTitle,omitempty"`
	AvatarURL string    `json:"avatarUrl"`
}

type BulkUserRequest struct {
	UserIDs []uuid.UUID `json:"userIds"`
}

type BulkUsernameRequest struct {
	Usernames []string `json:"usernames"`
}

type BulkUserResponse struct {
	Code    int    `json:"code"`
	Success bool   `json:"success"`
	Data    []User `json:"data"` // Changed to array
}
