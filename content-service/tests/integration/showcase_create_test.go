package integration

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/septianpadli/talas/content-service/internal/entity"
	"github.com/septianpadli/talas/content-service/tests/helper"
	"github.com/stretchr/testify/assert"
)

func TestCreateShowcase_Success(t *testing.T) {
	// 1. Setup App & DB
	app, db, mockPub := setupIntegrationAppWithMock()

	// 2. Get Valid Category ID
	var category entity.Category
	if err := db.First(&category).Error; err != nil {
		t.Fatalf("Failed to get category: %v", err)
	}
	categoryID := category.ID.String()

	// 3. Prepare Auth Token
	userID := uuid.New().String()
	token, err := helper.GenerateTestToken(userID)
	assert.NoError(t, err)

	// 4. Create Multipart Request
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	// Add Fields
	writer.WriteField("title", "Integration Test Project")
	writer.WriteField("content", "This is a content for integration testing purpose.")
	writer.WriteField("category_id", categoryID)
	writer.WriteField("tags", "test,integration")

	// Add File (Dummy Image)
	part, err := writer.CreateFormFile("files", "test_image.jpg")
	assert.NoError(t, err)
	part.Write([]byte("mock-image-content-bytes"))

	writer.Close()

	// 5. Create Request
	req := httptest.NewRequest("POST", "/api/showcases", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.AddCookie(&http.Cookie{Name: "access_token", Value: token})

	// 6. Execute
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)

	// 7. Assertions
	assert.Equal(t, 201, resp.StatusCode)

	var response map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&response)

	assert.Equal(t, true, response["success"])
	assert.Equal(t, "Showcase created successfully", response["message"])

	// Check Data validity in Response
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "Integration Test Project", data["title"])
	assert.Equal(t, categoryID, data["category_id"])
	
	// Extract Created ID
	createdID := data["id"].(string)

	// Check Database
	var showcase entity.Showcase
	err = db.Where("id = ?", createdID).First(&showcase).Error
	assert.NoError(t, err)
	assert.Equal(t, "Integration Test Project", showcase.Title)
	assert.Equal(t, categoryID, showcase.CategoryID.String())
	
	// Check Collaborator (Owner should be the creator)
	err = db.Model(&showcase).Association("Collaborators").Find(&showcase.Collaborators)
	assert.NoError(t, err)
	assert.Len(t, showcase.Collaborators, 1)
	assert.Equal(t, userID, showcase.Collaborators[0].UserID.String())
	assert.Equal(t, entity.CollaborationRoleOwner, showcase.Collaborators[0].Role)

	// Check RabbitMQ Event (Async)
	time.Sleep(50 * time.Millisecond) // Wait for goroutine
	
	assert.NotEmpty(t, mockPub.Events, "Event should be published")
	lastEvent := mockPub.Events[len(mockPub.Events)-1]
	assert.Equal(t, "showcase.created", lastEvent["routingKey"])
	
	// eventData IS the payload (Raw Map)
	eventData, ok := lastEvent["data"].(map[string]interface{})
	assert.True(t, ok, "Payload should be map")
	
	// showcase.ID is uuid.UUID.
	assert.Equal(t, showcase.ID, eventData["id"])
	assert.Equal(t, "Integration Test Project", eventData["title"])
}

func TestCreateShowcase_ValidationError(t *testing.T) {
	app, db := setupIntegrationApp()

	// Valid Token
	token, _ := helper.GenerateTestToken(uuid.New().String())

	// Get Valid Category
	var category entity.Category
	db.First(&category)
	categoryID := category.ID.String()

	// Multipart Request WITHOUT Title
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	// Missing Title field
	// writer.WriteField("title", "...") 
	writer.WriteField("content", "Content without title")
	writer.WriteField("category_id", categoryID)
	
	// Add File
	part, _ := writer.CreateFormFile("files", "test.jpg")
	part.Write([]byte("image"))
	
	writer.Close()

	req := httptest.NewRequest("POST", "/api/showcases", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.AddCookie(&http.Cookie{Name: "access_token", Value: token})

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)

	// Expect 400 Bad Request
	assert.Equal(t, 400, resp.StatusCode)
}
