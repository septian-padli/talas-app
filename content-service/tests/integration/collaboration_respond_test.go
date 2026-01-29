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

func TestRespondInvitation_Success(t *testing.T) {
	// 1. Setup
	app, db := setupIntegrationApp()

	// 2. Data: User A (Owner), User B (Invitee)
	userA := uuid.New()
	userB := uuid.New()
	
	// 3. Seed Showcase & Invitation
	// Get Category
	var category entity.Category
	db.First(&category)

	showcaseID := uuid.New()
	invitationID := uuid.New()

	showcase := entity.Showcase{
		Base:       entity.Base{ID: showcaseID},
		Title:      "Project Respond Success",
		Slug:       "project-respond-success-" + uuid.New().String(),
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
				Base:   entity.Base{ID: invitationID}, // Explicit Invitation ID
				UserID: userB,
				Role:   entity.CollaborationRoleCollaborator,
				Status: entity.CollaborationStatusPending,
			},
		},
	}
	err := db.Create(&showcase).Error
	assert.NoError(t, err)

	// 4. Token for User B (The Invitee)
	token, _ := helper.GenerateTestToken(userB.String())

	// 5. Request
	reqBody := map[string]string{
		"response": "ACCEPTED",
	}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/collaborations/%s/response", invitationID), bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "access_token", Value: token})

	// 6. Execute
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)

	// 7. Assertions
	assert.Equal(t, 200, resp.StatusCode)

	var response map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&response)
	assert.Equal(t, true, response["success"])

	// Check DB
	var invitation entity.Collaborator
	err = db.Where("id = ?", invitationID).First(&invitation).Error
	assert.NoError(t, err)
	assert.Equal(t, entity.CollaborationStatusAccepted, invitation.Status)
}

func TestRespondInvitation_Forbidden(t *testing.T) {
	// 1. Setup
	app, db := setupIntegrationApp()

	// 2. Data: User A (Owner), User B (Invitee), User C (Stranger)
	userA := uuid.New()
	userB := uuid.New()
	userC := uuid.New()
	
	// 3. Seed Showcase & Invitation for User B
	var category entity.Category
	db.First(&category)

	showcaseID := uuid.New()
	invitationID := uuid.New()

	showcase := entity.Showcase{
		Base:       entity.Base{ID: showcaseID},
		Title:      "Project Respond Forbidden",
		Slug:       "project-respond-forbidden-" + uuid.New().String(),
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
				Base:   entity.Base{ID: invitationID},
				UserID: userB, // Invite for B
				Role:   entity.CollaborationRoleCollaborator,
				Status: entity.CollaborationStatusPending,
			},
		},
	}
	db.Create(&showcase)

	// 4. Token for User C (The Stranger)
	token, _ := helper.GenerateTestToken(userC.String())

	// 5. Request
	reqBody := map[string]string{
		"response": "ACCEPTED",
	}
	jsonBody, _ := json.Marshal(reqBody)

	// User C tries to respond to User B's invitation
	req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/collaborations/%s/response", invitationID), bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "access_token", Value: token})

	// 6. Execute
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)

	// 7. Assert 403 Forbidden
	assert.Equal(t, 403, resp.StatusCode)
}
