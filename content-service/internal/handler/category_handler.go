package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/septianpadli/talas/content-service/internal/entity"
	"github.com/septianpadli/talas/content-service/internal/usecase"
)

type CategoryHandler struct {
	usecase usecase.CategoryUsecase
}

func NewCategoryHandler(usecase usecase.CategoryUsecase) *CategoryHandler {
	return &CategoryHandler{usecase: usecase}
}

// GET /categories?limit=20&cursor=uuid
func (h *CategoryHandler) GetCategories(c *fiber.Ctx) error {
	limitStr := c.Query("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 20
	}
	cursorStr := c.Query("cursor", "")
	var cursor uuid.UUID
	if cursorStr != "" {
		cursor, err = uuid.Parse(cursorStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid cursor"})
		}
	}
	categories, nextCursor, err := h.usecase.ListCategories(c.Context(), limit, cursor)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    500,
			"success": false,
			"message": err.Error(),
		})
	}
	// Mapping ke DTO tanpa deleted_at
	var categoryDTOs []entity.CategoryDTO
	for _, cat := range categories {
		categoryDTOs = append(categoryDTOs, entity.CategoryDTO{
			ID:        cat.ID.String(),
			Name:      cat.Name,
			Slug:      cat.Slug,
			CreatedAt: cat.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt: cat.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	hasNext := nextCursor != nil
	resp := fiber.Map{
		"code":   200,
		"status": true,
		"data": fiber.Map{
			"categories": categoryDTOs,
			"pagination": fiber.Map{
				"next_cursor": nil,
				"has_next":    hasNext,
			},
		},
	}
	if nextCursor != nil {
		resp["data"].(fiber.Map)["pagination"].(fiber.Map)["next_cursor"] = nextCursor.String()
	}
	return c.JSON(resp)
}

// GET /categories/:id?showcase=true&cursor=uuid&limit=10
func (h *CategoryHandler) GetCategoryDetail(c *fiber.Ctx) error {
	idParam := c.Params("id")
	var id uuid.UUID
	var slug string
	if uuidVal, err := uuid.Parse(idParam); err == nil {
		id = uuidVal
	} else {
		slug = idParam
	}

	showcaseParam := c.Query("showcase", "false")
	withShowcase := showcaseParam == "true"
	cursorStr := c.Query("cursor", "")
	var cursor uuid.UUID
	if cursorStr != "" {
		cursor, _ = uuid.Parse(cursorStr)
	}
	limitStr := c.Query("limit", "10")
	limit, _ := strconv.Atoi(limitStr)
	if limit < 1 {
		limit = 10
	}

	category, showcases, nextCursor, err := h.usecase.GetCategoryDetail(c.Context(), id, slug, withShowcase, cursor, limit)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"code":    404,
			"status":  false,
			"message": "Category not found",
		})
	}

	categoryDTO := entity.CategoryDTO{
		ID:        category.ID.String(),
		Name:      category.Name,
		Slug:      category.Slug,
		CreatedAt: category.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: category.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	showcasesDTO := []map[string]interface{}{}
	for _, s := range showcases {
		showcasesDTO = append(showcasesDTO, map[string]interface{}{
			"id":         s.ID.String(),
			"title":      s.Title,
			"created_at": s.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	var nextCursorVal interface{}
	if nextCursor != nil {
		nextCursorVal = nextCursor.String()
	} else {
		nextCursorVal = nil
	}

	categoryResp := map[string]interface{}{
		"id":         categoryDTO.ID,
		"name":       categoryDTO.Name,
		"slug":       categoryDTO.Slug,
		"created_at": categoryDTO.CreatedAt,
		"updated_at": categoryDTO.UpdatedAt,
		"showcases": map[string]interface{}{
			"items":       showcasesDTO,
			"next_cursor": nextCursorVal,
		},
	}

	if !withShowcase {
		categoryResp["showcases"] = map[string]interface{}{
			"items":       []interface{}{},
			"next_cursor": nil,
		}
	}

	return c.Status(200).JSON(fiber.Map{
		"code":   200,
		"status": true,
		"data":   categoryResp,
	})
}
