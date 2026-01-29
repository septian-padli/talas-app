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

func TestInviteCollaborator_Success(t *testing.T) {
	// 1. Setup App & DB
	app, db := setupIntegrationApp()

	// 2. Prepare Data: User A (Owner) & User B (Invitee)
	userA := uuid.New()
	// userB will be resolved by MockUserClient from username "user_b"

	// 3. Seed Showcase owned by User A
	// Need valid Category first
	var category entity.Category
	db.First(&category)

	showcase := entity.Showcase{
		Base:        entity.Base{ID: uuid.New()}, // Explicit ID
		Title:       "Project Alpha",
		Slug:        "project-alpha-" + uuid.New().String(),
		Content:     "Content for invitation test",
		CategoryID:  category.ID,
		Collaborators: []entity.Collaborator{
			{
				UserID: userA,
				Role:   entity.CollaborationRoleOwner,
				Status: entity.CollaborationStatusAccepted,
			},
		},
	}
	err := db.Create(&showcase).Error
	assert.NoError(t, err)

	// 4. Generate Token for User A
	token, _ := helper.GenerateTestToken(userA.String())

	// 5. Prepare Request Body
	reqBody := map[string]interface{}{
		"usernames": []string{"user_b"},
	}
	jsonBody, _ := json.Marshal(reqBody)

	// 6. Execute Request
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/showcases/%s/collaborators", showcase.ID), bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "access_token", Value: token})

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)

	// 7. Assertions
	assert.Equal(t, 200, resp.StatusCode)

	var response map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&response)
	
	assert.Equal(t, true, response["success"])

	// Check DB for new Collaborator
	var newCol entity.Collaborator
	// We don't know User B's UUID because Mock generates it randomly inside GetUsersByUsernames
	// However, we can query by ShowcaseID and Role=Collaborator
	err = db.Where("showcase_id = ? AND role = ?", showcase.ID, entity.CollaborationRoleCollaborator).First(&newCol).Error
	assert.NoError(t, err)
	assert.Equal(t, entity.CollaborationStatusPending, newCol.Status)
}

func TestInviteCollaborator_Forbidden(t *testing.T) {
	// 1. Setup
	app, db := setupIntegrationApp()

	// 2. Data: User A (Owner), User C (Stranger)
	userA := uuid.New()
	userC := uuid.New()

	// 3. Seed Showcase owned by User A
	var category entity.Category
	db.First(&category)

	showcase := entity.Showcase{
		Base:        entity.Base{ID: uuid.New()}, // Explicit ID
		Title:       "Project Beta",
		Slug:        "project-beta-" + uuid.New().String(),
		Content:     "Content for forbidden test",
		CategoryID:  category.ID,
		Collaborators: []entity.Collaborator{
			{
				UserID: userA,
				Role:   entity.CollaborationRoleOwner,
				Status: entity.CollaborationStatusAccepted,
			},
		},
	}
	db.Create(&showcase)

	// 4. Token for User C (Stranger)
	token, _ := helper.GenerateTestToken(userC.String())

	// 5. Request
	reqBody := map[string]interface{}{
		"usernames": []string{"user_any"},
	}
	jsonBody, _ := json.Marshal(reqBody)

	// 6. Execute
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/showcases/%s/collaborators", showcase.ID), bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "access_token", Value: token})

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)

	// 7. Assert 403 Forbidden
	assert.Equal(t, 403, resp.StatusCode)
}
