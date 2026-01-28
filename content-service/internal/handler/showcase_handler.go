package handler

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/septianpadli/talas/content-service/internal/entity"
	"github.com/septianpadli/talas/content-service/internal/usecase"
	"github.com/septianpadli/talas/content-service/pkg/utils"
)

type ShowcaseHandler struct {
	usecase usecase.ShowcaseUsecase
}

func NewShowcaseHandler(usecase usecase.ShowcaseUsecase) *ShowcaseHandler {
	return &ShowcaseHandler{usecase: usecase}
}

func (h *ShowcaseHandler) CreateShowcase(c *fiber.Ctx) error {
	var req entity.CreateShowcaseRequest

	// 1. Parse Form Body
	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, 400, "Invalid form data", nil)
	}

	// 2. Get Files
	form, err := c.MultipartForm()
	if err != nil {
		return utils.ErrorResponse(c, 400, "Invalid form data", nil)
	}
	files := form.File["files"] // Looking for 'files' key
	
	// Fallback to 'file' if 'files' is empty (backward compatibility/postman single file test)
	if len(files) == 0 {
		if fileHeader, err := c.FormFile("file"); err == nil {
			files = append(files, fileHeader)
		}
	}

	if len(files) == 0 {
		return utils.ErrorResponse(c, 400, "Image files are required", nil)
	}

	// 3. Get User ID from Context (Middleware)
	userIDStr, ok := c.Locals("user_id").(string)
	if !ok {
		return utils.ErrorResponse(c, 401, "Unauthorized: User ID missing", nil)
	}
	userID, _ := uuid.Parse(userIDStr)

	// 4. Call Usecase
	result, err := h.usecase.CreateShowcase(c.Context(), &req, files, userID)
	if err != nil {
		fmt.Printf("❌ CreateShowcase Error: %v\n", err) // Debug print
		
		// Map known validation errors to 400 Bad Request
		errMsg := err.Error()
		if contains(errMsg, "too large") || 
		   contains(errMsg, "invalid type") || 
		   contains(errMsg, "is required") ||
		   contains(errMsg, "Key:") { // Validator error usually starts with Key:
			return utils.ErrorResponse(c, 400, errMsg, nil)
		}

		return utils.ErrorResponse(c, 500, "Internal Server Error", nil)
	}

	return utils.SuccessResponse(c, 201, "Showcase created successfully", result)
}

func (h *ShowcaseHandler) GetShowcaseBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return utils.ErrorResponse(c, 400, "Slug is required", nil)
	}

	showcase, err := h.usecase.GetShowcaseBySlug(c.Context(), slug)
	if err != nil {
		if err.Error() == "record not found" {
			return utils.ErrorResponse(c, 404, "Showcase not found", nil)
		}
		return utils.ErrorResponse(c, 500, err.Error(), nil)
	}

	// Wrapper response to match contract
	// Contract: { code: 200, success: true, data: { showcase: ... } }
	return utils.SuccessResponse(c, 200, "Showcase found", fiber.Map{
		"showcase": showcase,
	})
}

func (h *ShowcaseHandler) GetShowcasesByUser(c *fiber.Ctx) error {
	userID := c.Params("id")
	cursor := c.Query("cursor")
	limit := c.QueryInt("limit", 10)

	result, err := h.usecase.GetShowcasesByUser(c.Context(), userID, limit, cursor)
	if err != nil {
		if err.Error() == "invalid user id format" {
			return utils.ErrorResponse(c, 400, "Invalid User ID", nil)
		}
		return utils.ErrorResponse(c, 500, err.Error(), nil)
	}

	return utils.SuccessResponse(c, 200, "User showcases retrieved", result)
}

func (h *ShowcaseHandler) GetMyShowcases(c *fiber.Ctx) error {
	// Get User ID from Middleware
	userIDVal := c.Locals("user_id")
	if userIDVal == nil {
		return utils.ErrorResponse(c, 401, "Unauthorized", nil)
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		// Try parsing if string
		if strID, ok := userIDVal.(string); ok {
			parsedID, err := uuid.Parse(strID)
			if err == nil {
				userID = parsedID
			} else {
				return utils.ErrorResponse(c, 401, "Invalid User ID in Token", nil)
			}
		} else {
			return utils.ErrorResponse(c, 401, "Invalid User ID type", nil)
		}
	}

	cursor := c.Query("cursor")
	limit := c.QueryInt("limit", 10)

	result, err := h.usecase.GetMyShowcases(c.Context(), userID, limit, cursor)
	if err != nil {
		return utils.ErrorResponse(c, 500, err.Error(), nil)
	}

	return utils.SuccessResponse(c, 200, "My showcases retrieved", result)
}

func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
