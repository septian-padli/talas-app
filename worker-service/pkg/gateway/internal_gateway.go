package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/septian/worker-service/internal/config"
)

type InternalGateway struct {
	client                *http.Client
	userServiceURL        string
	contentServiceURL     string
	internalServiceSecret string
}

func NewInternalGateway(cfg *config.Config) *InternalGateway {
	return &InternalGateway{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		userServiceURL:        cfg.UserServiceURL,
		contentServiceURL:     cfg.ContentServiceURL,
		internalServiceSecret: cfg.InternalServiceSecret,
	}
}

// User DTO for bulk user response
type User struct {
	ID         string `json:"id"`
	Username   string `json:"username"`
	Name       string `json:"name"`
	AvatarURL  string `json:"avatarUrl"`
	JobTitle   string `json:"jobTitle"`
	IsVerified bool   `json:"isVerified"`
}

// Showcase DTO for internal showcase response
type Showcase struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Slug    string `json:"slug"`
	OwnerID string `json:"owner_id"`
}

// GetUsersByIDs fetches bulk user data from user-service
func (g *InternalGateway) GetUsersByIDs(ctx context.Context, userIDs []string) (map[string]User, error) {
	// Prepare request body
	reqBody := map[string]interface{}{
		"userIds": userIDs,
	}
	bodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create request
	url := fmt.Sprintf("%s/api/internal/users/bulk", g.userServiceURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(bodyJSON)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-service-secret", g.internalServiceSecret)

	// Execute request
	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var response struct {
		Success bool   `json:"success"`
		Data    []User `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to map
	userMap := make(map[string]User)
	for _, user := range response.Data {
		userMap[user.ID] = user
	}

	return userMap, nil
}

// GetShowcaseByID fetches showcase data from content-service
func (g *InternalGateway) GetShowcaseByID(ctx context.Context, id string) (*Showcase, error) {
	// Create request
	url := fmt.Sprintf("%s/api/internal/showcases/%s", g.contentServiceURL, id)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.Header.Set("x-service-secret", g.internalServiceSecret)

	// Execute request
	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Handle 404 gracefully
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	// Check status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var response struct {
		Success bool      `json:"success"`
		Data    *Showcase `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Data, nil
}
