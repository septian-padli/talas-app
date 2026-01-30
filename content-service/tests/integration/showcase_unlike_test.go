package integration

import (
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

func TestToggleLike_Unlike(t *testing.T) {
	// 1. Setup
	app, db, mockPub := setupIntegrationAppWithMock()

	// 2. Data
	userA := uuid.New()
	userOwner := uuid.New()
	showcaseID := uuid.New()

	// 3. Seed Showcase with Existing Like
	var category entity.Category
	db.First(&category)

	showcase := entity.Showcase{
		Base:       entity.Base{ID: showcaseID},
		Title:      "Project to Unlike",
		Slug:       "project-to-unlike-" + uuid.New().String(),
		Content:    "Content",
		CategoryID: category.ID,
		Collaborators: []entity.Collaborator{
			{
				Base:   entity.Base{ID: uuid.New()},
				UserID: userOwner,
				Role:   entity.CollaborationRoleOwner,
				Status: entity.CollaborationStatusAccepted,
			},
		},
	}
	db.Create(&showcase)

	// Create existing like
	like := entity.ShowcaseLike{
		UserID:      userA,
		ShowcaseID:  showcaseID,
	}
	db.Create(&like)

	// 4. Token for User A
	token, _ := helper.GenerateTestToken(userA.String())

	// 5. Execute Request (Toggle to Unlike)
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/showcases/%s/like", showcaseID), nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: token})

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)

	// 6. Assertions
	assert.Equal(t, 200, resp.StatusCode)

	var response map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&response)

	assert.Equal(t, true, response["success"])
	data := response["data"].(map[string]interface{})
	assert.Equal(t, false, data["is_liked"]) // Should be unliked now

	// Check DB (Like should be removed)
	var likeCount int64
	db.Model(&entity.ShowcaseLike{}).Where("user_id = ? AND showcase_id = ?", userA, showcaseID).Count(&likeCount)
	assert.Equal(t, int64(0), likeCount)

	// 7. Verify RabbitMQ Event
	time.Sleep(50 * time.Millisecond)
	
	assert.NotEmpty(t, mockPub.Events, "Event should be published")
	lastEvent := mockPub.Events[len(mockPub.Events)-1]
	assert.Equal(t, "showcase.unliked", lastEvent["routingKey"])
	
	eventData, ok := lastEvent["data"].(map[string]interface{})
	assert.True(t, ok)
	
	assert.Equal(t, showcaseID, eventData["showcase_id"])
	assert.Equal(t, userA, eventData["actor_id"])
	assert.Equal(t, userOwner, eventData["target_user_id"])
}
