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

func TestReplyComment_Success(t *testing.T) {
	// 1. Setup
	app, db, mockPub := setupIntegrationAppWithMock()

	// 2. Data
	userA := uuid.New() // Author of Root
	userB := uuid.New() // Author of Reply
	showcaseID := uuid.New()
	rootCommentID := uuid.New()

	// 3. Seed Showcase & Root Comment
	var category entity.Category
	db.First(&category)

	showcase := entity.Showcase{
		Base:       entity.Base{ID: showcaseID},
		Title:      "Project for Reply",
		Slug:       "project-reply-" + uuid.New().String(),
		Content:    "Content",
		CategoryID: category.ID,
		Collaborators: []entity.Collaborator{
			{UserID: userA, Role: entity.CollaborationRoleOwner, Status: entity.CollaborationStatusAccepted},
		},
	}
	db.Create(&showcase)

	rootComment := entity.Comment{
		Base:       entity.Base{ID: rootCommentID},
		ShowcaseID: showcaseID,
		UserID:     userA,
		Body:       "This is root comment",
	}
	db.Create(&rootComment)

	// 4. Token for User B
	token, _ := helper.GenerateTestToken(userB.String())

	// 5. Request
	reqBody := map[string]interface{}{
		"content": "This is a reply",
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", fmt.Sprintf("/api/comments/%s/reply", rootCommentID), bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "access_token", Value: token})

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)

	// 6. Assertions
	assert.Equal(t, 201, resp.StatusCode)

	var response map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&response)
	assert.Equal(t, true, response["success"])

	data := response["data"].(map[string]interface{})
	commentData := data["comment"].(map[string]interface{})

	// Check Response Data
	assert.Equal(t, "This is a reply", commentData["body"])
	assert.Equal(t, showcaseID.String(), commentData["showcase_id"])
	assert.Equal(t, rootCommentID.String(), commentData["parent_id"])

	// Check DB
	var replyDB entity.Comment
	// Find by Body and User
	err = db.Where("user_id = ? AND body = ?", userB, "This is a reply").First(&replyDB).Error
	assert.NoError(t, err)
	
	// Check inheritance
	assert.Equal(t, showcaseID, replyDB.ShowcaseID)
	assert.NotNil(t, replyDB.ParentID)
	assert.Equal(t, rootCommentID, *replyDB.ParentID)

	// 7. Verify RabbitMQ Event
	time.Sleep(50 * time.Millisecond)
	
	assert.NotEmpty(t, mockPub.Events, "Event should be published")
	lastEvent := mockPub.Events[len(mockPub.Events)-1]
	assert.Equal(t, "comment.replied", lastEvent["routingKey"])
	
	eventData, ok := lastEvent["data"].(map[string]interface{})
	assert.True(t, ok)
	
	assert.Equal(t, replyDB.ID, eventData["reply_id"])
	assert.Equal(t, rootCommentID, eventData["parent_id"])
	assert.Equal(t, showcaseID, eventData["showcase_id"])
	assert.Equal(t, userB, eventData["actor_id"])
	assert.Equal(t, userA, eventData["target_user_id"]) // Parent Author
	assert.Equal(t, userA, eventData["showcase_owner_id"]) // Showcase Owner
}

func TestReplyComment_FailedNotFound(t *testing.T) {
	// 1. Setup
	app, _ := setupIntegrationApp()

	// 2. Data
	userB := uuid.New()
	ghostID := uuid.New()

	// 3. Token
	token, _ := helper.GenerateTestToken(userB.String())

	// 4. Request
	reqBody := map[string]interface{}{"content": "Reply to ghost"}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", fmt.Sprintf("/api/comments/%s/reply", ghostID), bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "access_token", Value: token})

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)

	// 5. Assertions
	assert.Equal(t, 404, resp.StatusCode)
}
