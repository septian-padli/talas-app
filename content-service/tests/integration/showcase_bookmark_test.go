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

func TestToggleBookmark_Bookmark(t *testing.T) {
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
		Title:      "Project to Bookmark",
		Slug:       "project-to-bookmark-" + uuid.New().String(),
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
	db.Create(&showcase)

	// Ensure NO existing bookmark
	db.Where("user_id = ? AND showcase_id = ?", userA, showcaseID).Delete(&entity.Bookmark{})

	// 4. Token for User A
	token, _ := helper.GenerateTestToken(userA.String())

	// 5. Execute Request
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/showcases/%s/bookmark", showcaseID), nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: token})

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)

	// 6. Assertions
	assert.Equal(t, 200, resp.StatusCode)

	var response map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&response)

	assert.Equal(t, true, response["success"])
	data := response["data"].(map[string]interface{})
	assert.Equal(t, true, data["is_bookmarked"])

	// Check DB
	var bookmarkCount int64
	db.Model(&entity.Bookmark{}).Where("user_id = ? AND showcase_id = ?", userA, showcaseID).Count(&bookmarkCount)
	assert.Equal(t, int64(1), bookmarkCount)

	// 7. Verify RabbitMQ Event
	time.Sleep(50 * time.Millisecond)
	
	assert.NotEmpty(t, mockPub.Events, "Event should be published")
	lastEvent := mockPub.Events[len(mockPub.Events)-1]
	assert.Equal(t, "showcase.bookmarked", lastEvent["routingKey"])
	
	eventData, ok := lastEvent["data"].(map[string]interface{})
	assert.True(t, ok)
	
	assert.Equal(t, showcaseID, eventData["showcase_id"])
	assert.Equal(t, userA, eventData["actor_id"])
}

func TestToggleBookmark_Unbookmark(t *testing.T) {
	// 1. Setup
	app, db, mockPub := setupIntegrationAppWithMock()

	// 2. Data
	userA := uuid.New()
	showcaseID := uuid.New()

	// 3. Seed Showcase with Existing Bookmark
	var category entity.Category
	db.First(&category)

	showcase := entity.Showcase{
		Base:       entity.Base{ID: showcaseID},
		Title:      "Project to Unbookmark",
		Slug:       "project-to-unbookmark-" + uuid.New().String(),
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
	db.Create(&showcase)

	// Create existing bookmark
	bookmark := entity.Bookmark{
		UserID:     userA,
		ShowcaseID: showcaseID,
	}
	db.Create(&bookmark)

	// 4. Token for User A
	token, _ := helper.GenerateTestToken(userA.String())

	// 5. Execute Request (Toggle to Unbookmark)
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/showcases/%s/bookmark", showcaseID), nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: token})

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)

	// 6. Assertions
	assert.Equal(t, 200, resp.StatusCode)

	var response map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&response)

	assert.Equal(t, true, response["success"])
	data := response["data"].(map[string]interface{})
	assert.Equal(t, false, data["is_bookmarked"]) // Should be unbookmarked

	// Check DB (Bookmark should be removed)
	var bookmarkCount int64
	db.Model(&entity.Bookmark{}).Where("user_id = ? AND showcase_id = ?", userA, showcaseID).Count(&bookmarkCount)
	assert.Equal(t, int64(0), bookmarkCount)

	// 7. Verify RabbitMQ Event
	time.Sleep(50 * time.Millisecond)
	
	assert.NotEmpty(t, mockPub.Events, "Event should be published")
	lastEvent := mockPub.Events[len(mockPub.Events)-1]
	assert.Equal(t, "showcase.unbookmarked", lastEvent["routingKey"])
	
	eventData, ok := lastEvent["data"].(map[string]interface{})
	assert.True(t, ok)
	
	assert.Equal(t, showcaseID, eventData["showcase_id"])
	assert.Equal(t, userA, eventData["actor_id"])
}
