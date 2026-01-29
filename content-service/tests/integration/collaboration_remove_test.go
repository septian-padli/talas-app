package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/septianpadli/talas/content-service/internal/entity"
	"github.com/septianpadli/talas/content-service/tests/helper"
	"github.com/stretchr/testify/assert"
)

func TestRemoveCollaborator_SuccessKick(t *testing.T) {
	// 1. Setup
	app, db := setupIntegrationApp()

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
