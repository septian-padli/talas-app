package clients

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/septianpadli/talas/content-service/internal/config"
	"github.com/sirupsen/logrus"
)

type UserClient interface {
	GetUsersBulk(userIDs []uuid.UUID) (map[uuid.UUID]UserDetail, error)
	GetUsersByUsernames(usernames []string) (map[string]UserDetail, error)
}

type userClient struct {
	cfg        *config.Config
	httpClient *http.Client
	log        *logrus.Logger
}

type UserDetail struct {
	ID        uuid.UUID `json:"id"` // Added ID field
	Name      string    `json:"name"`
	Username  string    `json:"username"`
	AvatarURL string    `json:"avatarUrl"` // Changed to avatarUrl to match User Service
}

type bulkUserRequest struct {
	UserIDs []uuid.UUID `json:"userIds"`
}

type bulkUsernameRequest struct {
	Usernames []string `json:"usernames"`
}

type bulkUserResponse struct {
	Code    int          `json:"code"`
	Success bool         `json:"success"`
	Data    []UserDetail `json:"data"` // Changed to array
}

func NewUserClient(cfg *config.Config, log *logrus.Logger) UserClient {
	return &userClient{
		cfg: cfg,
		log: log,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *userClient) GetUsersBulk(userIDs []uuid.UUID) (map[uuid.UUID]UserDetail, error) {
	url := fmt.Sprintf("%s/api/internal/users/bulk", c.cfg.UserServiceURL)

	reqBody := bulkUserRequest{UserIDs: userIDs}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-service-secret", c.cfg.InternalServiceSecret)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.log.Errorf("Failed to call user service: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.log.Warnf("User service returned status: %d", resp.StatusCode)
		return nil, fmt.Errorf("user service error: %d", resp.StatusCode)
	}

	var apiResp bulkUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("user service returned failure")
	}

	// Convert Array to Map for easier lookup
	result := make(map[uuid.UUID]UserDetail)
	for _, user := range apiResp.Data {
		result[user.ID] = user
	}

	return result, nil
}

func (c *userClient) GetUsersByUsernames(usernames []string) (map[string]UserDetail, error) {
	url := fmt.Sprintf("%s/api/internal/users/lookup", c.cfg.UserServiceURL)

	reqBody := bulkUsernameRequest{Usernames: usernames}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-service-secret", c.cfg.InternalServiceSecret)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.log.Errorf("Failed to call user service (lookup): %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.log.Warnf("User service lookup returned status: %d", resp.StatusCode)
		return nil, fmt.Errorf("user service error: %d", resp.StatusCode)
	}

	var apiResp bulkUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("user service returned failure")
	}

	// Convert Array to Map (Key: Username)
	result := make(map[string]UserDetail)
	for _, user := range apiResp.Data {
		result[user.Username] = user
	}

	return result, nil
}
