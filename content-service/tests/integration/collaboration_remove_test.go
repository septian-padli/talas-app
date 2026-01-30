package integration

import (
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

func TestRemoveCollaborator_SuccessKick(t *testing.T) {
	// 1. Setup
	app, db, mockPub := setupIntegrationAppWithMock()

	// 2. Data: User A (Owner), User B (Member)
	userA := uuid.New()
	userB := uuid.New()

	// 3. Seed Showcase with Owner A and Member B
	var category entity.Category
	db.First(&category)

	showcaseID := uuid.New()

	showcase := entity.Showcase{
		Base:       entity.Base{ID: showcaseID},
		Title:      "Project Kick Success",
		Slug:       "project-kick-success-" + uuid.New().String(),
		Content:    "Content",
		CategoryID: category.ID,
		Collaborators: []entity.Collaborator{
			{
				Base:   entity.Base{ID: uuid.New()},
				UserID: userA,
				Role:   entity.CollaborationRoleOwner,
				Status: entity.CollaborationStatusAccepted,
			},
			{
				Base:   entity.Base{ID: uuid.New()},
				UserID: userB,
				Role:   entity.CollaborationRoleCollaborator,
				Status: entity.CollaborationStatusAccepted,
			},
		},
	}
	err := db.Create(&showcase).Error
	assert.NoError(t, err)

	// 4. Token for Owner A
	token, _ := helper.GenerateTestToken(userA.String())

	// 5. Execute Kick Request (A kicks B)
	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/showcases/%s/collaborators/%s", showcaseID, userB), nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: token})

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)

	// 6. Assertions
	assert.Equal(t, 200, resp.StatusCode)

	// Verify DB: User B should be gone
	var count int64
	db.Model(&entity.Collaborator{}).Where("showcase_id = ? AND user_id = ?", showcaseID, userB).Count(&count)
	assert.Equal(t, int64(0), count, "Collaborator should be removed from DB")

	// 8. Verify RabbitMQ Event (Kick)
	time.Sleep(100 * time.Millisecond)
	
	assert.NotEmpty(t, mockPub.Events, "Event should be published")
	lastEvent := mockPub.Events[len(mockPub.Events)-1]
	assert.Equal(t, "collaborator.removed", lastEvent["routingKey"])
	
	eventData, ok := lastEvent["data"].(map[string]interface{})
	assert.True(t, ok)
	
	assert.Equal(t, showcaseID, eventData["showcase_id"])
	assert.Equal(t, "Project Kick Success", eventData["showcase_title"])
	assert.Equal(t, userA, eventData["actor_id"])       // Owner kicked
	assert.Equal(t, userB, eventData["target_user_id"]) // Member removed
	assert.Equal(t, false, eventData["is_self_removal"])
	assert.Equal(t, userA, eventData["showcase_owner_id"])
}

func TestRemoveCollaborator_SuccessLeave(t *testing.T) {
	// 1. Setup
	app, db, mockPub := setupIntegrationAppWithMock()

	// 2. Data: User A (Owner), User B (Member)
	userA := uuid.New()
	userB := uuid.New()

	// 3. Seed Showcase
	var category entity.Category
	db.First(&category)

	showcaseID := uuid.New()

	showcase := entity.Showcase{
		Base:       entity.Base{ID: showcaseID},
		Title:      "Project Leave Success",
		Slug:       "project-leave-success-" + uuid.New().String(),
		Content:    "Content",
		CategoryID: category.ID,
		Collaborators: []entity.Collaborator{
			{
				Base:   entity.Base{ID: uuid.New()},
				UserID: userA,
				Role:   entity.CollaborationRoleOwner,
				Status: entity.CollaborationStatusAccepted,
			},
			{
				Base:   entity.Base{ID: uuid.New()},
				UserID: userB,
				Role:   entity.CollaborationRoleCollaborator,
				Status: entity.CollaborationStatusAccepted,
			},
		},
	}
	db.Create(&showcase)

	// 4. Token for Member B (Leaving)
	token, _ := helper.GenerateTestToken(userB.String())

	// 5. Execute Leave Request (B removes B)
	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/showcases/%s/collaborators/%s", showcaseID, userB), nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: token})

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)

	// 6. Assertions
	assert.Equal(t, 200, resp.StatusCode)

	// Verify Event (Leave)
	time.Sleep(100 * time.Millisecond)
	
	assert.NotEmpty(t, mockPub.Events, "Event should be published")
	lastEvent := mockPub.Events[len(mockPub.Events)-1]
	assert.Equal(t, "collaborator.removed", lastEvent["routingKey"])
	
	eventData, ok := lastEvent["data"].(map[string]interface{})
	assert.True(t, ok)
	
	assert.Equal(t, showcaseID, eventData["showcase_id"])
	assert.Equal(t, "Project Leave Success", eventData["showcase_title"])
	assert.Equal(t, userB, eventData["actor_id"])       // Member left
	assert.Equal(t, userB, eventData["target_user_id"]) // Member removed
	assert.Equal(t, true, eventData["is_self_removal"])
	assert.Equal(t, userA, eventData["showcase_owner_id"]) // Owner ID remains
}

func TestRemoveCollaborator_FailedKickOwner(t *testing.T) {
	// 1. Setup
	app, db := setupIntegrationApp()

	// 2. Data: User A (Owner), User B (Member)
	userA := uuid.New()
	userB := uuid.New()

	// 3. Seed Showcase
	var category entity.Category
	db.First(&category)

	showcaseID := uuid.New()

	showcase := entity.Showcase{
		Base:       entity.Base{ID: showcaseID},
		Title:      "Project Kick Fail",
		Slug:       "project-kick-fail-" + uuid.New().String(),
		Content:    "Content",
		CategoryID: category.ID,
		Collaborators: []entity.Collaborator{
			{
				Base:   entity.Base{ID: uuid.New()},
				UserID: userA,
				Role:   entity.CollaborationRoleOwner,
				Status: entity.CollaborationStatusAccepted,
			},
			{
				Base:   entity.Base{ID: uuid.New()},
				UserID: userB, // Member
				Role:   entity.CollaborationRoleCollaborator,
				Status: entity.CollaborationStatusAccepted,
			},
		},
	}
	db.Create(&showcase)

	// 4. Token for Member B
	token, _ := helper.GenerateTestToken(userB.String())

	// 5. Execute Kick Request (Member B tries to kick Owner A)
	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/showcases/%s/collaborators/%s", showcaseID, userA), nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: token})

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)

	// 6. Assertions
	assert.Equal(t, 403, resp.StatusCode)
}
