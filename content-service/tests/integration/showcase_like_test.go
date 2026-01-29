package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/septianpadli/talas/content-service/internal/entity"
	"github.com/septianpadli/talas/content-service/tests/helper"
	"github.com/stretchr/testify/assert"
)

func TestToggleLike_Success(t *testing.T) {
	// 1. Setup
	app, db := setupIntegrationApp()

	// 2. Data
	userA := uuid.New()
	showcaseID := uuid.New()

	// 3. Seed Showcase
	var category entity.Category
	db.First(&category)

	showcase := entity.Showcase{
		Base:       entity.Base{ID: showcaseID},
		Title:      "Project to Like",
		Slug:       "project-to-like-" + uuid.New().String(),
		Content:    "Content",
		CategoryID: category.ID,
		Collaborators: []entity.Collaborator{
			{
				Base:   entity.Base{ID: uuid.New()},
				UserID: uuid.New(), // Owned by random person
				Role:   entity.CollaborationRoleOwner,
				Status: entity.CollaborationStatusAccepted,
			},
		},
	}
	err := db.Create(&showcase).Error
	assert.NoError(t, err)

	// Ensure NO existing like
	db.Where("user_id = ? AND showcase_id = ?", userA, showcaseID).Delete(&entity.ShowcaseLike{})

	// 4. Token for User A
	token, _ := helper.GenerateTestToken(userA.String())

	// 5. Execute Request
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/showcases/%s/like", showcaseID), nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: token})

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)

	// 6. Assertions
	assert.Equal(t, 200, resp.StatusCode)

	var response map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&response)

	assert.Equal(t, true, response["success"])
	
	// Check data payload: {"is_liked": true} 
	data := response["data"].(map[string]interface{})
	assert.Equal(t, true, data["is_liked"])

	// Check DB
	var likeCount int64
	db.Model(&entity.ShowcaseLike{}).Where("user_id = ? AND showcase_id = ?", userA, showcaseID).Count(&likeCount)
	assert.Equal(t, int64(1), likeCount)
}

func TestToggleLike_NotFound(t *testing.T) {
	// 1. Setup
	app, _ := setupIntegrationApp()

	// 2. Data
	userA := uuid.New()
	ghostID := uuid.New() // Non-existent showcase

	// 3. Token
	token, _ := helper.GenerateTestToken(userA.String())

	// 4. Execute
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/showcases/%s/like", ghostID), nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: token})

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)

	// 5. Assertions
	assert.Equal(t, 404, resp.StatusCode)
}
