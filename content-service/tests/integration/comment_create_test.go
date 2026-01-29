package integration

import (
	"bytes"
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

func TestCreateComment_SuccessRoot(t *testing.T) {
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
		Title:      "Project for Commenting",
		Slug:       "project-commenting-" + uuid.New().String(),
		Content:    "Content",
		CategoryID: category.ID,
		Collaborators: []entity.Collaborator{
			{
				Base:   entity.Base{ID: uuid.New()},
				UserID: uuid.New(),
				Role:   entity.CollaborationRoleOwner,
				Status: entity.CollaborationStatusAccepted,
			},
		},
	}
	err := db.Create(&showcase).Error
	assert.NoError(t, err)

	// 4. Token for User A
	token, _ := helper.GenerateTestToken(userA.String())

	// 5. Request Body
	reqBody := map[string]interface{}{
		"content": "Project yang bagus",
	}
	jsonBody, _ := json.Marshal(reqBody)

	// 6. Execute Request
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/showcases/%s/comments", showcaseID), bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "access_token", Value: token})

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)

	// 7. Assertions
	assert.Equal(t, 201, resp.StatusCode)

	var response map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&response)

	assert.Equal(t, true, response["success"])
	
	// Check Data Payload
	data := response["data"].(map[string]interface{})
	commentData := data["comment"].(map[string]interface{})
	assert.Equal(t, "Project yang bagus", commentData["body"])
	assert.Equal(t, showcaseID.String(), commentData["showcase_id"])
	assert.Nil(t, commentData["parent_id"]) // Should be nil for root

	// Check DB
	var commentDB entity.Comment
	err = db.Where("showcase_id = ? AND user_id = ? AND body = ?", showcaseID, userA, "Project yang bagus").First(&commentDB).Error
	assert.NoError(t, err)
	assert.Nil(t, commentDB.ParentID)
}

func TestCreateComment_FailedShowcaseNotFound(t *testing.T) {
	// 1. Setup
	app, _ := setupIntegrationApp()

	// 2. Data
	userA := uuid.New()
	ghostID := uuid.New()

	// 3. Token
	token, _ := helper.GenerateTestToken(userA.String())

	// 4. Request Body
	reqBody := map[string]interface{}{
		"content": "Komentar Nyasar",
	}
	jsonBody, _ := json.Marshal(reqBody)

	// 5. Execute
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/showcases/%s/comments", ghostID), bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "access_token", Value: token})

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)

	// 6. Assertions
	assert.Equal(t, 404, resp.StatusCode)
}
