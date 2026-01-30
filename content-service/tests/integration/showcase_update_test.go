package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/septianpadli/talas/content-service/internal/entity"
	"github.com/septianpadli/talas/content-service/tests/helper"
	"github.com/stretchr/testify/assert"
)

func TestUpdateShowcase_Success(t *testing.T) {
	// 1. Setup
	app, db, mockPub := setupIntegrationAppWithMock()

	// 2. Data
	userA := uuid.New()
	showcaseID := uuid.New()

	// 3. Seed Showcase
	var category entity.Category
	db.First(&category)

	showcase := entity.Showcase{
		Base:       entity.Base{ID: showcaseID},
		Title:      "Old Title",
		Slug:       "old-title-" + uuid.New().String(),
		Content:    "Content",
		CategoryID: category.ID,
		Collaborators: []entity.Collaborator{
			{UserID: userA, Role: entity.CollaborationRoleOwner, Status: entity.CollaborationStatusAccepted},
		},
	}
	db.Create(&showcase)

	// 4. Token for User A
	token, _ := helper.GenerateTestToken(userA.String())

	// 5. Request Body
	reqBody := map[string]interface{}{
		"title": "New Title",
	}
	jsonBody, _ := json.Marshal(reqBody)

	// 6. Execute
	req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/showcases/%s", showcaseID), bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "access_token", Value: token})

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)

	// 7. Assertions
	assert.Equal(t, 200, resp.StatusCode)

	// 8. Verify DB
	var showcaseDB entity.Showcase
	err = db.Where("id = ?", showcaseID).First(&showcaseDB).Error
	assert.NoError(t, err)
	assert.Equal(t, "New Title", showcaseDB.Title)
	// Check updated_at is recent (within last minute)
	assert.True(t, showcaseDB.UpdatedAt.After(time.Now().Add(-1*time.Minute)))

	// 9. Verify RabbitMQ Event
	time.Sleep(50 * time.Millisecond) // Wait for async publish
	
	assert.NotEmpty(t, mockPub.Events, "Event should be published")
	lastEvent := mockPub.Events[len(mockPub.Events)-1]
	assert.Equal(t, "showcase.updated", lastEvent["routingKey"])
	
	// eventData IS the payload
	eventData, ok := lastEvent["data"].(map[string]interface{})
	assert.True(t, ok)
	
	assert.Equal(t, showcaseID, eventData["id"])
	assert.Equal(t, "New Title", eventData["title"])
}

func TestUpdateShowcase_Forbidden(t *testing.T) {
	// 1. Setup
	app, db := setupIntegrationApp()

	// 2. Data
	userA := uuid.New() // Owner
	userB := uuid.New() // Attacker
	showcaseID := uuid.New()

	// 3. Seed
	var category entity.Category
	db.First(&category)
	showcase := entity.Showcase{
		Base:       entity.Base{ID: showcaseID},
		Title:      "Original Title",
		Slug:       "orig-title-" + uuid.New().String(),
		Content:    "Content",
		CategoryID: category.ID,
		Collaborators: []entity.Collaborator{
			{UserID: userA, Role: entity.CollaborationRoleOwner, Status: entity.CollaborationStatusAccepted},
		},
	}
	db.Create(&showcase)

	// 4. Token for User B
	token, _ := helper.GenerateTestToken(userB.String())

	// 5. Request
	reqBody := map[string]interface{}{"title": "Hacked Title"}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/showcases/%s", showcaseID), bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "access_token", Value: token})

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)

	// 6. Assertions
	assert.Equal(t, 403, resp.StatusCode)

	// 7. Verify DB Unchanged
	var showcaseDB entity.Showcase
	db.Where("id = ?", showcaseID).First(&showcaseDB)
	assert.Equal(t, "Original Title", showcaseDB.Title)
}
