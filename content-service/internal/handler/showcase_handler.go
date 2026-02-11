package handler

import (
	"strings"

	"mime/multipart"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/septianpadli/talas/content-service/internal/entity"
	"github.com/septianpadli/talas/content-service/internal/usecase"
	"github.com/septianpadli/talas/content-service/pkg/utils"
	"github.com/sirupsen/logrus"
)

type ShowcaseHandler struct {
	usecase usecase.ShowcaseUsecase
	log     *logrus.Logger
}

func NewShowcaseHandler(usecase usecase.ShowcaseUsecase, log *logrus.Logger) *ShowcaseHandler {
	return &ShowcaseHandler{
		usecase: usecase,
		log:     log,
	}
}

func (h *ShowcaseHandler) CreateShowcase(c *fiber.Ctx) error {
	// Request ID from Middleware
	reqID, _ := c.Locals("requestid").(string)

	var req entity.CreateShowcaseRequest

	// 1. Parse Form Body
	if err := c.BodyParser(&req); err != nil {
		h.log.WithFields(logrus.Fields{
			"request_id": reqID,
			"error":      err.Error(),
		}).Error("CreateShowcase: Invalid form data")
		return utils.ErrorResponse(c, 400, "Invalid form data", nil)
	}

	// 2. Get Files
	form, err := c.MultipartForm()
	if err != nil {
		h.log.WithFields(logrus.Fields{
			"request_id": reqID,
			"error":      err.Error(),
		}).Error("CreateShowcase: Failed to parse multipart form")
		return utils.ErrorResponse(c, 400, "Invalid form data", nil)
	}
	files := form.File["files"]
	if len(files) == 0 {
		if fileHeader, err := c.FormFile("file"); err == nil {
			files = append(files, fileHeader)
		}
	}

	if len(files) == 0 {
		h.log.WithFields(logrus.Fields{
			"request_id": reqID,
		}).Error("CreateShowcase: No files provided")
		return utils.ErrorResponse(c, 400, "Image files are required", nil)
	}

	// 3. Get User ID from Context
	userIDStr, ok := c.Locals("user_id").(string)
	if !ok {
		h.log.WithFields(logrus.Fields{
			"request_id": reqID,
		}).Error("CreateShowcase: Unauthorized (User ID missing)")
		return utils.ErrorResponse(c, 401, "Unauthorized: User ID missing", nil)
	}
	userID, _ := uuid.Parse(userIDStr)

	// LOG: Process Started
	h.log.WithFields(logrus.Fields{
		"request_id": reqID,
		"user_id":    userID.String(),
		"title":      req.Title, // Assuming Title is in request
	}).Info("Create showcase process started")

	// 4. Call Usecase
	result, err := h.usecase.CreateShowcase(c.Context(), &req, files, userID)
	if err != nil {
		errMsg := err.Error()

		// Log detailed error
		h.log.WithFields(logrus.Fields{
			"request_id": reqID,
			"user_id":    userID.String(),
			"error":      errMsg,
		}).Error("CreateShowcase: Usecase execution failed")

		if utils.Contains(errMsg, "too large") ||
			utils.Contains(errMsg, "invalid type") ||
			utils.Contains(errMsg, "is required") ||
			utils.Contains(errMsg, "Key:") {
			return utils.ErrorResponse(c, 400, errMsg, nil)
		}

		return utils.ErrorResponse(c, 500, "Internal Server Error", nil)
	}

	// LOG: Success
	h.log.WithFields(logrus.Fields{
		"request_id":  reqID,
		"user_id":     userID.String(),
		"showcase_id": result.ID.String(),
		"slug":        result.Slug,
	}).Info("Showcase created successfully")

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

func (h *ShowcaseHandler) GetTrendingFeeds(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 10)
	cursor := c.Query("cursor")

	result, err := h.usecase.GetTrendingFeeds(c.Context(), limit, cursor)
	if err != nil {
		h.log.Errorf("GetTrendingFeeds Error: %v", err)
		return utils.ErrorResponse(c, 500, "Failed to fetch trending feeds", nil)
	}

	return utils.SuccessResponse(c, 200, "Trending feeds retrieved", result)
}

func (h *ShowcaseHandler) UpdateShowcase(c *fiber.Ctx) error {
	// Request ID for Tracing
	reqID, _ := c.Locals("requestid").(string)

	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.log.WithFields(logrus.Fields{
			"request_id":  reqID,
			"showcase_id": idStr,
			"error":       err.Error(),
		}).Error("UpdateShowcase: Invalid UUID format")
		return utils.ErrorResponse(c, 400, "Invalid UUID format", nil)
	}

	var req entity.UpdateShowcaseRequest
	if err := c.BodyParser(&req); err != nil {
		h.log.WithFields(logrus.Fields{
			"request_id":  reqID,
			"showcase_id": id.String(),
			"error":       err.Error(),
		}).Error("UpdateShowcase: Invalid JSON body")
		return utils.ErrorResponse(c, 400, "Invalid JSON body", nil)
	}

	// Parsing User ID from Locals
	userIDVal := c.Locals("user_id")
	if userIDVal == nil {
		h.log.WithFields(logrus.Fields{
			"request_id":  reqID,
			"showcase_id": id.String(),
		}).Error("UpdateShowcase: Unauthorized (User ID missing)")
		return utils.ErrorResponse(c, 401, "Unauthorized", nil)
	}

	var userID uuid.UUID
	if str, ok := userIDVal.(string); ok {
		userID, _ = uuid.Parse(str)
	} else if uid, ok := userIDVal.(uuid.UUID); ok {
		userID = uid
	} else {
		h.log.WithFields(logrus.Fields{
			"request_id":  reqID,
			"showcase_id": id.String(),
		}).Error("UpdateShowcase: Invalid User ID context")
		return utils.ErrorResponse(c, 401, "Invalid User ID context", nil)
	}

	// LOG: Process Started
	h.log.WithFields(logrus.Fields{
		"request_id":  reqID,
		"user_id":     userID.String(),
		"showcase_id": id.String(),
		"payload_keys": func() []string {
			// Helper to log what fields are being updated (Metadata)
			var keys []string
			if req.Title != nil {
				keys = append(keys, "title")
			}
			if req.Content != nil {
				keys = append(keys, "content")
			}
			if req.CategoryID != nil {
				keys = append(keys, "category_id")
			}
			if req.Tags != nil {
				keys = append(keys, "tags")
			}
			return keys
		}(),
	}).Info("Update showcase process started")

	// Check for Empty Body (Warn)
	if req.Title == nil && req.Content == nil && req.CategoryID == nil && req.Tags == nil {
		h.log.WithFields(logrus.Fields{
			"request_id":  reqID,
			"user_id":     userID.String(),
			"showcase_id": id.String(),
		}).Warn("UpdateShowcase: Empty update request body")
	}

	// Get Files (optional, for update)
	var files []*multipart.FileHeader
	form, err := c.MultipartForm()
	if err == nil && form != nil {
		files = form.File["files"]
	}
	if len(files) == 0 {
		if fileHeader, err := c.FormFile("file"); err == nil {
			files = append(files, fileHeader)
		}
	}

	result, err := h.usecase.UpdateShowcase(c.Context(), id, &req, userID, files)
	if err != nil {
		errMsg := err.Error()

		logFields := logrus.Fields{
			"request_id":  reqID,
			"user_id":     userID.String(),
			"showcase_id": id.String(),
			"error":       errMsg,
		}

		if strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "record not found") {
			h.log.WithFields(logFields).Error("UpdateShowcase: Showcase not found")
			return utils.ErrorResponse(c, 404, "Showcase not found", nil)
		}
		if strings.Contains(errMsg, "forbidden") {
			// LOG: Warn Forbidden (Potential Attack)
			h.log.WithFields(logFields).Warn("UpdateShowcase: Forbidden access attempt")
			return utils.ErrorResponse(c, 403, "Forbidden", nil)
		}
		if strings.Contains(errMsg, "content must be") || strings.Contains(errMsg, "invalid category") || strings.Contains(errMsg, "title must be") {
			h.log.WithFields(logFields).Error("UpdateShowcase: Validation failed")
			return utils.ErrorResponse(c, 400, errMsg, nil)
		}

		// LOG: Fatal/Technical Error (We use Error level but treat as critical)
		h.log.WithFields(logFields).Error("UpdateShowcase: Technical/Database error")
		return utils.ErrorResponse(c, 500, "Internal Server Error", nil)
	}

	// LOG: Success
	h.log.WithFields(logrus.Fields{
		"request_id":  reqID,
		"user_id":     userID.String(),
		"showcase_id": result.ID.String(),
		"slug":        result.Slug,
	}).Info("Showcase updated successfully")

	return utils.SuccessResponse(c, 200, "Showcase updated successfully", result)
}

func (h *ShowcaseHandler) ToggleLike(c *fiber.Ctx) error {
	// Request ID for Tracing
	reqID, _ := c.Locals("requestid").(string)

	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.log.WithFields(logrus.Fields{
			"request_id":  reqID,
			"showcase_id": idStr,
			"error":       err.Error(),
		}).Error("ToggleLike: Invalid UUID format")
		return utils.ErrorResponse(c, 400, "Invalid UUID format", nil)
	}

	userIDVal := c.Locals("user_id")
	if userIDVal == nil {
		h.log.WithFields(logrus.Fields{
			"request_id":  reqID,
			"showcase_id": id.String(),
		}).Error("ToggleLike: Unauthorized (User ID missing)")
		return utils.ErrorResponse(c, 401, "Unauthorized", nil)
	}

	var userID uuid.UUID
	if str, ok := userIDVal.(string); ok {
		userID, _ = uuid.Parse(str)
	} else if uid, ok := userIDVal.(uuid.UUID); ok {
		userID = uid
	} else {
		h.log.WithFields(logrus.Fields{
			"request_id":  reqID,
			"showcase_id": id.String(),
		}).Error("ToggleLike: Invalid User ID context")
		return utils.ErrorResponse(c, 401, "Invalid User ID context", nil)
	}

	// LOG: Process Started
	h.log.WithFields(logrus.Fields{
		"request_id":  reqID,
		"user_id":     userID.String(),
		"showcase_id": id.String(),
		"action":      "like",
	}).Info("Toggle like action started")

	isLiked, err := h.usecase.ToggleLike(c.Context(), userID, id)
	if err != nil {
		logFields := logrus.Fields{
			"request_id":  reqID,
			"user_id":     userID.String(),
			"showcase_id": id.String(),
			"error":       err.Error(),
		}

		if strings.Contains(err.Error(), "not found") {
			// LOG: WARN - Resource Not Found
			h.log.WithFields(logFields).Warn("ToggleLike: Showcase not found")
			return utils.ErrorResponse(c, 404, "Showcase not found", nil)
		}

		// LOG: ERROR - Database/Logic Error
		h.log.WithFields(logFields).Error("ToggleLike: Database error")
		return utils.ErrorResponse(c, 500, err.Error(), nil)
	}

	msg := "Showcase unliked"
	action := "unliked"
	if isLiked {
		msg = "Showcase liked"
		action = "liked"
	}

	// LOG: Success
	h.log.WithFields(logrus.Fields{
		"request_id":  reqID,
		"user_id":     userID.String(),
		"showcase_id": id.String(),
		"status":      action,
	}).Info("Toggle like action completed")

	return utils.SuccessResponse(c, 200, msg, fiber.Map{"is_liked": isLiked})
}

func (h *ShowcaseHandler) ToggleBookmark(c *fiber.Ctx) error {
	// Request ID for Tracing
	reqID, _ := c.Locals("requestid").(string)

	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.log.WithFields(logrus.Fields{
			"request_id":  reqID,
			"showcase_id": idStr,
			"error":       err.Error(),
		}).Error("ToggleBookmark: Invalid UUID format")
		return utils.ErrorResponse(c, 400, "Invalid UUID format", nil)
	}

	userIDVal := c.Locals("user_id")
	if userIDVal == nil {
		h.log.WithFields(logrus.Fields{
			"request_id":  reqID,
			"showcase_id": id.String(),
		}).Error("ToggleBookmark: Unauthorized (User ID missing)")
		return utils.ErrorResponse(c, 401, "Unauthorized", nil)
	}

	var userID uuid.UUID
	if str, ok := userIDVal.(string); ok {
		userID, _ = uuid.Parse(str)
	} else if uid, ok := userIDVal.(uuid.UUID); ok {
		userID = uid
	} else {
		h.log.WithFields(logrus.Fields{
			"request_id":  reqID,
			"showcase_id": id.String(),
		}).Error("ToggleBookmark: Invalid User ID context")
		return utils.ErrorResponse(c, 401, "Invalid User ID context", nil)
	}

	// LOG: Process Started
	h.log.WithFields(logrus.Fields{
		"request_id":  reqID,
		"user_id":     userID.String(),
		"showcase_id": id.String(),
		"action":      "bookmark",
	}).Info("Toggle bookmark action started")

	isBookmarked, err := h.usecase.ToggleBookmark(c.Context(), userID, id)
	if err != nil {
		logFields := logrus.Fields{
			"request_id":  reqID,
			"user_id":     userID.String(),
			"showcase_id": id.String(),
			"error":       err.Error(),
		}

		if strings.Contains(err.Error(), "not found") {
			// LOG: WARN - Resource Not Found
			h.log.WithFields(logFields).Warn("ToggleBookmark: Showcase not found")
			return utils.ErrorResponse(c, 404, "Showcase not found", nil)
		}

		// LOG: ERROR - Database/Logic Error
		h.log.WithFields(logFields).Error("ToggleBookmark: Database error")
		return utils.ErrorResponse(c, 500, err.Error(), nil)
	}

	msg := "Showcase unbookmarked"
	action := "unbookmarked"
	if isBookmarked {
		msg = "Showcase bookmarked"
		action = "bookmarked"
	}

	// LOG: Success
	h.log.WithFields(logrus.Fields{
		"request_id":  reqID,
		"user_id":     userID.String(),
		"showcase_id": id.String(),
		"status":      action,
	}).Info("Toggle bookmark action completed")

	return utils.SuccessResponse(c, 200, msg, fiber.Map{"is_bookmarked": isBookmarked})
}

// DeleteShowcase handles deleting a showcase
func (h *ShowcaseHandler) DeleteShowcase(c *fiber.Ctx) error {
	// Request ID for Tracing
	reqID, _ := c.Locals("requestid").(string)

	showcaseID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		h.log.WithFields(logrus.Fields{
			"request_id":  reqID,
			"showcase_id": c.Params("id"),
			"error":       err.Error(),
		}).Error("DeleteShowcase: Invalid UUID format")
		return utils.ErrorResponse(c, 400, "Invalid showcase UUID", nil)
	}

	userIDVal := c.Locals("user_id")
	if userIDVal == nil {
		h.log.WithFields(logrus.Fields{
			"request_id":  reqID,
			"showcase_id": showcaseID.String(),
		}).Error("DeleteShowcase: Unauthorized (User ID missing)")
		return utils.ErrorResponse(c, 401, "Unauthorized", nil)
	}

	var userID uuid.UUID
	if str, ok := userIDVal.(string); ok {
		userID, _ = uuid.Parse(str)
	} else if uid, ok := userIDVal.(uuid.UUID); ok {
		userID = uid
	} else {
		h.log.WithFields(logrus.Fields{
			"request_id":  reqID,
			"showcase_id": showcaseID.String(),
		}).Error("DeleteShowcase: Invalid User ID context")
		return utils.ErrorResponse(c, 401, "Invalid User ID context", nil)
	}

	// LOG: Process Started
	h.log.WithFields(logrus.Fields{
		"request_id":  reqID,
		"user_id":     userID.String(),
		"showcase_id": showcaseID.String(),
	}).Info("Delete showcase process started")

	if err := h.usecase.DeleteShowcase(c.Context(), showcaseID, userID); err != nil {
		errMsg := err.Error()

		logFields := logrus.Fields{
			"request_id":  reqID,
			"user_id":     userID.String(),
			"showcase_id": showcaseID.String(),
			"error":       errMsg,
		}

		if strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "record not found") {
			// LOG: WARN - Resource Not Found
			h.log.WithFields(logFields).Warn("DeleteShowcase: Resource not found")
			return utils.ErrorResponse(c, 404, errMsg, nil)
		}
		if strings.Contains(errMsg, "forbidden") {
			// LOG: WARN - Unauthorized Access Attempt
			h.log.WithFields(logFields).Warn("DeleteShowcase: Forbidden access attempt")
			return utils.ErrorResponse(c, 403, errMsg, nil)
		}

		// LOG: ERROR - Database/Technical Error
		h.log.WithFields(logFields).Error("DeleteShowcase: Database or cleanup error")
		return utils.ErrorResponse(c, 500, "Internal Server Error", nil)
	}

	// LOG: Success
	h.log.WithFields(logrus.Fields{
		"request_id":  reqID,
		"user_id":     userID.String(),
		"showcase_id": showcaseID.String(),
	}).Info("Showcase deleted successfully")

	return utils.SuccessResponse(c, 200, "Showcase successfully deleted (archived)", nil)
}

func (h *ShowcaseHandler) SearchShowcases(c *fiber.Ctx) error {
	query := c.Query("q")
	limit := c.QueryInt("limit", 10)
	cursor := c.Query("cursor")
	categorySlugs := c.Query("category") // Comma separated
	sortBy := c.Query("sort")            // latest, popular, relevance

	result, err := h.usecase.SearchShowcases(c.Context(), query, limit, cursor, categorySlugs, sortBy)
	if err != nil {
		if err.Error() == "invalid cursor format" {
			return utils.ErrorResponse(c, 400, "Invalid cursor format", nil)
		}
		return utils.ErrorResponse(c, 500, err.Error(), nil)
	}

	return utils.SuccessResponse(c, 200, "Search results retrieved", result)
}
