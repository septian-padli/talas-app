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

func TestDeleteShowcase_SuccessSoft(t *testing.T) {
	// 1. Setup
	app, db := setupIntegrationApp()

	// 2. Data
	userA := uuid.New()
	showcaseID := uuid.New()

	// 3. Seed
	var category entity.Category
	db.First(&category)
	showcase := entity.Showcase{
		Base:       entity.Base{ID: showcaseID},
		Title:      "To Be Deleted",
		Slug:       "to-be-del-" + uuid.New().String(),
		Content:    "Content",
		CategoryID: category.ID,
		Collaborators: []entity.Collaborator{
			{UserID: userA, Role: entity.CollaborationRoleOwner, Status: entity.CollaborationStatusAccepted},
		},
	}
	db.Create(&showcase)

	// Ensure not deleted
	assert.False(t, showcase.DeletedAt.Valid)

	// 4. Token
	token, _ := helper.GenerateTestToken(userA.String())

	// 5. Request
	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/showcases/%s", showcaseID), nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: token})

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)

	// 6. Assertions
	assert.Equal(t, 200, resp.StatusCode)

	// 7. Verify DB Soft Delete
	var showcaseDB entity.Showcase
	// Use Unscoped to find soft-deleted records
	err = db.Unscoped().Where("id = ?", showcaseID).First(&showcaseDB).Error
	assert.NoError(t, err)
	
	assert.True(t, showcaseDB.DeletedAt.Valid, "Showcase should be soft deleted")
	assert.True(t, showcaseDB.DeletedAt.Time.After(time.Now().Add(-1*time.Minute)))
}

func TestDeleteShowcase_Forbidden(t *testing.T) {
	// 1. Setup
	app, db := setupIntegrationApp()

	// 2. Data
	userA := uuid.New()
	userB := uuid.New()
	showcaseID := uuid.New()

	// 3. Seed
	var category entity.Category
	db.First(&category)
	showcase := entity.Showcase{
		Base:       entity.Base{ID: showcaseID},
		Title:      "Protected Project",
		Slug:       "protected-proj-" + uuid.New().String(),
		Content:    "Content",
		CategoryID: category.ID,
		Collaborators: []entity.Collaborator{
			{UserID: userA, Role: entity.CollaborationRoleOwner, Status: entity.CollaborationStatusAccepted},
		},
	}
	db.Create(&showcase)

	// 4. Token User B
	token, _ := helper.GenerateTestToken(userB.String())

	// 5. Request
	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/showcases/%s", showcaseID), nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: token})

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)

	// 6. Assertions
	assert.Equal(t, 403, resp.StatusCode)

	// 7. Verify DB
	var showcaseDB entity.Showcase
	err = db.Where("id = ?", showcaseID).First(&showcaseDB).Error
	assert.NoError(t, err)
	assert.False(t, showcaseDB.DeletedAt.Valid)
}
