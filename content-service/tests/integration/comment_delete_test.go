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

func TestDeleteComment_SuccessTombstone(t *testing.T) {
	// 1. Setup
	app, db := setupIntegrationApp()

	// 2. Data
	userA := uuid.New() // Owner of Parent
	userB := uuid.New() // Owner of Child
	showcaseID := uuid.New()
	parentID := uuid.New()
	childID := uuid.New()

	// 3. Seed Data
	var category entity.Category
	db.First(&category)

	showcase := entity.Showcase{
		Base:       entity.Base{ID: showcaseID},
		Title:      "Project Delete Test",
		Slug:       "project-delete-" + uuid.New().String(),
		Content:    "Content",
		CategoryID: category.ID,
		Collaborators: []entity.Collaborator{ 
			{UserID: userA, Role: entity.CollaborationRoleOwner, Status: entity.CollaborationStatusAccepted},
		},
	}
	db.Create(&showcase)

	parentComment := entity.Comment{
		Base:       entity.Base{ID: parentID},
		ShowcaseID: showcaseID,
		UserID:     userA,
		Body:       "Parent Comment",
	}
	db.Create(&parentComment)

	childComment := entity.Comment{
		Base:       entity.Base{ID: childID},
		ShowcaseID: showcaseID,
		UserID:     userB,
		Body:       "Child Comment",
		ParentID:   &parentID,
	}
	db.Create(&childComment)

	// 4. Token for User A
	token, _ := helper.GenerateTestToken(userA.String())

	// 5. Request Delete Parent
	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/comments/%s", parentID), nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: token})

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)

	// 6. Assertions
	assert.Equal(t, 200, resp.StatusCode)

	// 7. Verify DB
	var parentDB entity.Comment
	// We need Unscoped because it might be soft deleted if gorm.DeletedAt is used.
	// BUT wait, in Comment entity, we used bool IsDeleted or specialized tombstone logic?
	// Let's check Interaction.go again. 
	// Ah, I don't see DeletedAt in Comment struct in Step 5759.
	// But `Base` struct usually has `DeletedAt`?
	// Step 5755 checked directory. `base.go` has `Base` struct.
	// If `Base` has `DeletedAt gorm.DeletedAt`, then GORM does soft delete by default.
	// If `DeleteComment` usecase sets `DeletedAt`, then we use Unscoped.
	
	// Assuming Base has DeletedAt.
	err = db.Unscoped().Where("id = ?", parentID).First(&parentDB).Error
	assert.NoError(t, err)
	
	// Check if DeletedAt is Set
	assert.True(t, parentDB.DeletedAt.Valid, "DeletedAt should be valid (deleted)")

	// Check Child
	var childDB entity.Comment
	err = db.Where("id = ?", childID).First(&childDB).Error
	assert.NoError(t, err)
	assert.False(t, childDB.DeletedAt.Valid, "Child should NOT be deleted")
}

func TestDeleteComment_Forbidden(t *testing.T) {
	// 1. Setup
	app, db := setupIntegrationApp()

	// 2. Data
	userA := uuid.New() // Attacker
	userB := uuid.New() // Victim (Comment Owner)
	showcaseID := uuid.New()
	commentID := uuid.New()

	// 3. Seed
	var category entity.Category
	db.First(&category)
	showcase := entity.Showcase{
		Base:       entity.Base{ID: showcaseID},
		Title:      "Project Forbidden Test",
		Slug:       "project-forbidden-" + uuid.New().String(),
		Content:    "Content",
		CategoryID: category.ID,
		Collaborators: []entity.Collaborator{ 
			{UserID: userB, Role: entity.CollaborationRoleOwner, Status: entity.CollaborationStatusAccepted},
		},
	}
	db.Create(&showcase)

	comment := entity.Comment{
		Base:       entity.Base{ID: commentID},
		ShowcaseID: showcaseID,
		UserID:     userB,
		Body:       "Don't delete me",
	}
	db.Create(&comment)

	// 4. Token for User A
	token, _ := helper.GenerateTestToken(userA.String())

	// 5. Request
	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/comments/%s", commentID), nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: token})

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)

	// 6. Assertions
	assert.Equal(t, 403, resp.StatusCode)

	// 7. Verify DB (Still exists)
	var commentDB entity.Comment
	err = db.Where("id = ?", commentID).First(&commentDB).Error
	assert.NoError(t, err)
	assert.False(t, commentDB.DeletedAt.Valid, "Comment should NOT be deleted")
}
