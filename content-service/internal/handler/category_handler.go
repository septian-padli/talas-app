package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/septianpadli/talas/content-service/internal/entity"
	"github.com/septianpadli/talas/content-service/internal/usecase"
	"github.com/septianpadli/talas/content-service/pkg/utils"
)

// CreateCategoryRequest DTO
type CreateCategoryRequest struct {
	Name string `json:"name" validate:"required,max=100"`
}

type CategoryHandler struct {
	usecase usecase.CategoryUsecase
}

// POST /categories
var formatTimeStamp = "2006-01-02T15:04:05Z07:00"

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
			return utils.ErrorResponse(c, 400, "Invalid cursor", nil)
		}
	}
	categories, nextCursor, err := h.usecase.ListCategories(c.Context(), limit, cursor)
	if err != nil {
		return utils.ErrorResponse(c, 500, err.Error(), nil)
	}
	// Mapping ke DTO tanpa deleted_at
	var categoryDTOs []entity.CategoryDTO
	for _, cat := range categories {
		categoryDTOs = append(categoryDTOs, entity.CategoryDTO{
			ID:        cat.ID.String(),
			Name:      cat.Name,
			Slug:      cat.Slug,
			CreatedAt: cat.CreatedAt.Format(formatTimeStamp),
			UpdatedAt: cat.UpdatedAt.Format(formatTimeStamp),
		})
	}
	hasNext := nextCursor != nil
	pagination := fiber.Map{
		"next_cursor": nil,
		"has_next":    hasNext,
	}
	if nextCursor != nil {
		pagination["next_cursor"] = nextCursor.String()
	}
	data := fiber.Map{
		"categories": categoryDTOs,
		"pagination": pagination,
	}
	return utils.SuccessResponse(c, 200, "Categories retrieved successfully", data)
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
		return utils.ErrorResponse(c, 404, "Category not found", nil)
	}

	categoryDTO := entity.CategoryDTO{
		ID:        category.ID.String(),
		Name:      category.Name,
		Slug:      category.Slug,
		CreatedAt: category.CreatedAt.Format(formatTimeStamp),
		UpdatedAt: category.UpdatedAt.Format(formatTimeStamp),
	}

	showcasesDTO := []entity.ShowcaseDTOCategory{}
	for _, s := range showcases {
		showcasesDTO = append(showcasesDTO, entity.ShowcaseDTOCategory{
			ID:        s.ID.String(),
			Title:     s.Title,
			Slug:      s.Slug,
			Media:     s.Media,
			CreatedAt: s.CreatedAt.Format(formatTimeStamp),
		})
	}

	var nextCursorVal interface{}
	if nextCursor != nil {
		nextCursorVal = nextCursor.String()
	} else {
		nextCursorVal = nil
	}

	hasNext := nextCursor != nil
	pagination := map[string]interface{}{
		"has_next":    hasNext,
		"next_cursor": nextCursorVal,
	}

	categoryResp := map[string]interface{}{
		"id":         categoryDTO.ID,
		"name":       categoryDTO.Name,
		"slug":       categoryDTO.Slug,
		"created_at": categoryDTO.CreatedAt,
		"updated_at": categoryDTO.UpdatedAt,
		"showcases": map[string]interface{}{
			"items":      showcasesDTO,
			"pagination": pagination,
		},
	}

	if !withShowcase {
		categoryResp["showcases"] = map[string]interface{}{
			"items": []interface{}{},
			"pagination": map[string]interface{}{
				"has_next":    false,
				"next_cursor": nil,
			},
		}
	}

	return utils.SuccessResponse(c, 200, "Category detail retrieved successfully", categoryResp)
}

func (h *CategoryHandler) CreateCategory(c *fiber.Ctx) error {
	var req CreateCategoryRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, 400, "Invalid request body", nil)
	}
	// Validasi name (required, max 100)
	if len(req.Name) == 0 || len(req.Name) > 100 {
		return utils.ErrorResponse(c, 400, "Name is required and max 100 characters", nil)
	}
	// Akan dipanggil usecase selanjutnya
	category, err := h.usecase.CreateCategory(c.Context(), req.Name)
	if err != nil {
		if err.Error() == "category already exists" {
			return utils.ErrorResponse(c, 409, "Category sudah ada", nil)
		}
		return utils.ErrorResponse(c, 500, err.Error(), nil)
	}
	data := fiber.Map{
		"id":         category.ID.String(),
		"name":       category.Name,
		"slug":       category.Slug,
		"created_at": category.CreatedAt.Format(formatTimeStamp),
		"updated_at": category.UpdatedAt.Format(formatTimeStamp),
	}
	return utils.SuccessResponse(c, 201, "Category created successfully", data)
}
