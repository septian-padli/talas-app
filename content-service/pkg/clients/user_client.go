package clients

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/septianpadli/talas/content-service/internal/config"
	"github.com/septianpadli/talas/content-service/internal/entity"
	"github.com/sirupsen/logrus"
)

type UserClient interface {
	GetUsersBulk(userIDs []uuid.UUID) (map[uuid.UUID]*entity.User, error)
	GetUsersByUsernames(usernames []string) (map[string]*entity.User, error)
}

type userClient struct {
	cfg        *config.Config
	httpClient *http.Client
	log        *logrus.Logger
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

func (c *userClient) GetUsersBulk(userIDs []uuid.UUID) (map[uuid.UUID]*entity.User, error) {
	url := fmt.Sprintf("%s/api/internal/users/bulk", c.cfg.UserServiceURL)

	reqBody := entity.BulkUserRequest{UserIDs: userIDs}
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

	var apiResp entity.BulkUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("user service returned failure")
	}

	// Convert Array to Map for easier lookup
	result := make(map[uuid.UUID]*entity.User)
	// for _, user := range apiResp.Data {
	// 	result[user.ID] = user
	// }
	for i := range apiResp.Data {
		user := &apiResp.Data[i]
		result[user.ID] = user
	}

	return result, nil
}

func (c *userClient) GetUsersByUsernames(usernames []string) (map[string]*entity.User, error) {
	url := fmt.Sprintf("%s/api/internal/users/lookup", c.cfg.UserServiceURL)

	reqBody := entity.BulkUsernameRequest{Usernames: usernames}
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

	var apiResp entity.BulkUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("user service returned failure")
	}

	// Convert Array to Map (Key: Username)
	result := make(map[string]*entity.User)
	// for _, user := range apiResp.Data {
	// 	result[user.Username] = user
	// }
	for i := range apiResp.Data {
		user := &apiResp.Data[i]
		result[user.Username] = user
	}

	return result, nil
}
