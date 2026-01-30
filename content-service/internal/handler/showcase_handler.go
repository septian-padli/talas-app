package handler

import (
	"fmt"
	"strings"

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

		if contains(errMsg, "too large") || 
		   contains(errMsg, "invalid type") || 
		   contains(errMsg, "is required") ||
		   contains(errMsg, "Key:") { 
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


func (h *ShowcaseHandler) UpdateShowcase(c *fiber.Ctx) error {
	// Request ID for Tracing
	reqID, _ := c.Locals("requestid").(string)
	
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.log.WithFields(logrus.Fields{
			"request_id": reqID,
			"showcase_id": idStr,
			"error":      err.Error(),
		}).Error("UpdateShowcase: Invalid UUID format")
		return utils.ErrorResponse(c, 400, "Invalid UUID format", nil)
	}

	var req entity.UpdateShowcaseRequest
	if err := c.BodyParser(&req); err != nil {
		h.log.WithFields(logrus.Fields{
			"request_id": reqID,
			"showcase_id": id.String(),
			"error":      err.Error(),
		}).Error("UpdateShowcase: Invalid JSON body")
		return utils.ErrorResponse(c, 400, "Invalid JSON body", nil)
	}

	// Parsing User ID from Locals
	userIDVal := c.Locals("user_id")
	if userIDVal == nil {
		h.log.WithFields(logrus.Fields{
			"request_id": reqID,
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
			"request_id": reqID,
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
			if req.Title != nil { keys = append(keys, "title") }
			if req.Content != nil { keys = append(keys, "content") }
			if req.CategoryID != nil { keys = append(keys, "category_id") }
			if req.Tags != nil { keys = append(keys, "tags") }
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

	result, err := h.usecase.UpdateShowcase(c.Context(), id, &req, userID)
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


func (h *ShowcaseHandler) CreateComment(c *fiber.Ctx) error {
	// Request ID for Tracing
	reqID, _ := c.Locals("requestid").(string)

	idStr := c.Params("id")
	showcaseID, err := uuid.Parse(idStr)
	if err != nil {
		h.log.WithFields(logrus.Fields{
			"request_id":  reqID,
			"showcase_id": idStr,
			"error":       err.Error(),
		}).Error("CreateComment: Invalid Showcase UUID")
		return utils.ErrorResponse(c, 400, "Invalid Showcase UUID", nil)
	}

	userIDVal := c.Locals("user_id")
	if userIDVal == nil {
		h.log.WithFields(logrus.Fields{
			"request_id":  reqID,
			"showcase_id": showcaseID.String(),
		}).Error("CreateComment: Unauthorized (User ID missing)")
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
		}).Error("CreateComment: Invalid User ID context")
		return utils.ErrorResponse(c, 401, "Invalid User ID context", nil)
	}

	var req entity.CreateCommentRequest
	if err := c.BodyParser(&req); err != nil {
		h.log.WithFields(logrus.Fields{
			"request_id":  reqID,
			"showcase_id": showcaseID.String(),
			"error":       err.Error(),
		}).Error("CreateComment: Invalid JSON body")
		return utils.ErrorResponse(c, 400, "Invalid JSON body", nil)
	}

	if req.Content == "" {
		return utils.ErrorResponse(c, 400, "Content required", nil)
	}

	// Determine if New Discussion or Reply
	msgStart := "Create comment process started"
	typeKey := "new_discussion"
	if req.ParentID != nil {
		msgStart = "Create reply process started"
		typeKey = "reply"
	}

	// LOG: Process Started
	h.log.WithFields(logrus.Fields{
		"request_id":  reqID,
		"user_id":     userID.String(),
		"showcase_id": showcaseID.String(),
		"parent_id":   req.ParentID,
		"type":        typeKey,
	}).Info(msgStart)

	comment, err := h.usecase.CreateComment(c.Context(), showcaseID, &req, userID)
	if err != nil {
		errMsg := err.Error()
		logFields := logrus.Fields{
			"request_id":  reqID,
			"user_id":     userID.String(),
			"showcase_id": showcaseID.String(),
			"parent_id":   req.ParentID,
			"error":       errMsg,
		}

		if strings.Contains(errMsg, "showcase not found") || strings.Contains(errMsg, "parent comment not found") || strings.Contains(errMsg, "record not found") {
			// LOG: ERROR - Resource Not Found
			h.log.WithFields(logFields).Error("CreateComment: Resource not found")
			return utils.ErrorResponse(c, 404, errMsg, nil)
		}
		if strings.Contains(errMsg, "belong to invalid showcase") || strings.Contains(errMsg, "different showcase") {
			// LOG: WARN - Invalid Parent (Mismatch Showcase)
			h.log.WithFields(logFields).Warn("CreateComment: Invalid parent comment (showcase mismatch)")
			return utils.ErrorResponse(c, 400, errMsg, nil)
		}
		if strings.Contains(errMsg, "depth limit") || strings.Contains(errMsg, "maximum depth") {
			// LOG: WARN - Depth Limit
			h.log.WithFields(logFields).Warn("CreateComment: Recursion depth limit exceeded")
			return utils.ErrorResponse(c, 400, errMsg, nil)
		}
		if strings.Contains(errMsg, "content must be") || strings.Contains(errMsg, "invalid") {
			// Logic Error / Validation
			h.log.WithFields(logFields).Error("CreateComment: Validation failed")
			return utils.ErrorResponse(c, 400, errMsg, nil)
		}
		
		// LOG: ERROR - Database/Technical Error
		h.log.WithFields(logFields).Error("CreateComment: Database error")
		return utils.ErrorResponse(c, 500, "Internal Server Error", nil)
	}

	// LOG: Success
	h.log.WithFields(logrus.Fields{
		"request_id":  reqID,
		"user_id":     userID.String(),
		"showcase_id": showcaseID.String(),
		"comment_id":  comment.ID.String(),
		"parent_id":   comment.ParentID,
	}).Info("Comment/Reply created successfully")

	return utils.SuccessResponse(c, 201, "Comment created successfully", fiber.Map{
		"comment": comment,
	})
}

func (h *ShowcaseHandler) ReplyComment(c *fiber.Ctx) error {
	// Request ID for Tracing
	reqID, _ := c.Locals("requestid").(string)

	idStr := c.Params("id")
	parentID, err := uuid.Parse(idStr)
	if err != nil {
		h.log.WithFields(logrus.Fields{
			"request_id": reqID,
			"parent_id":  idStr,
			"error":      err.Error(),
		}).Error("ReplyComment: Invalid Parent Comment UUID")
		return utils.ErrorResponse(c, 400, "Invalid Parent Comment UUID", nil)
	}

	userIDVal := c.Locals("user_id")
	if userIDVal == nil {
		h.log.WithFields(logrus.Fields{
			"request_id": reqID,
			"parent_id":  parentID.String(),
		}).Error("ReplyComment: Unauthorized (User ID missing)")
		return utils.ErrorResponse(c, 401, "Unauthorized", nil)
	}
	var userID uuid.UUID
	if str, ok := userIDVal.(string); ok {
		userID, _ = uuid.Parse(str)
	} else if uid, ok := userIDVal.(uuid.UUID); ok {
		userID = uid
	} else {
		h.log.WithFields(logrus.Fields{
			"request_id": reqID,
			"parent_id":  parentID.String(),
		}).Error("ReplyComment: Invalid User ID context")
		return utils.ErrorResponse(c, 401, "Invalid User ID context", nil)
	}

	var req entity.ReplyCommentRequest
	if err := c.BodyParser(&req); err != nil {
		h.log.WithFields(logrus.Fields{
			"request_id": reqID,
			"parent_id":  parentID.String(),
			"error":      err.Error(),
		}).Error("ReplyComment: Invalid JSON body")
		return utils.ErrorResponse(c, 400, "Invalid JSON body", nil)
	}

	if req.Content == "" {
		return utils.ErrorResponse(c, 400, "Content required", nil)
	}

	// LOG: Process Started
	h.log.WithFields(logrus.Fields{
		"request_id": reqID,
		"user_id":    userID.String(),
		"parent_id":  parentID.String(),
	}).Info("Reply comment process started")

	comment, err := h.usecase.ReplyComment(c.Context(), parentID, &req, userID)
	if err != nil {
		errMsg := err.Error()
		logFields := logrus.Fields{
			"request_id": reqID,
			"user_id":    userID.String(),
			"parent_id":  parentID.String(),
			"error":      errMsg,
		}

		if strings.Contains(errMsg, "parent comment not found") || strings.Contains(errMsg, "record not found") {
			// LOG: ERROR - Parent Not Found
			h.log.WithFields(logFields).Error("ReplyComment: Parent comment not found")
			return utils.ErrorResponse(c, 404, errMsg, nil)
		}
		if strings.Contains(errMsg, "content must be") || strings.Contains(errMsg, "invalid") || strings.Contains(errMsg, "belong") {
			// LOG: ERROR - Validation/Logic
			h.log.WithFields(logFields).Error("ReplyComment: Validation failed")
			return utils.ErrorResponse(c, 400, errMsg, nil)
		}
		// Assuming usecase checks depth or spam, map generic Logic errors if present.
		if strings.Contains(errMsg, "depth limit") {
			// LOG: WARN - Nested Depth Warning
			h.log.WithFields(logFields).Warn("ReplyComment: Nested depth limit reached")
			return utils.ErrorResponse(c, 400, errMsg, nil)
		}
		
		// LOG: ERROR - Database/Technical Error
		h.log.WithFields(logFields).Error("ReplyComment: Database error")
		return utils.ErrorResponse(c, 500, "Internal Server Error", nil)
	}

	// LOG: Success
	h.log.WithFields(logrus.Fields{
		"request_id":  reqID,
		"user_id":     userID.String(),
		"parent_id":   parentID.String(),
		"reply_id":    comment.ID.String(),
		"showcase_id": comment.ShowcaseID.String(),
	}).Info("Reply created successfully")

	return utils.SuccessResponse(c, 201, "Comment replied successfully", fiber.Map{
		"comment": comment,
	})
}

func (h *ShowcaseHandler) GetShowcaseComments(c *fiber.Ctx) error {
	idStr := c.Params("id")
	showcaseID, err := uuid.Parse(idStr)
	if err != nil {
		return utils.ErrorResponse(c, 400, "Invalid Showcase UUID", nil)
	}

	limit := c.QueryInt("limit", 20)
	cursor := c.Query("cursor")

	result, err := h.usecase.GetShowcaseComments(c.Context(), showcaseID, limit, cursor)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "failed to fetch comments") {
			return utils.ErrorResponse(c, 500, "Internal Server Error", nil)
		}
		return utils.ErrorResponse(c, 500, "Internal Server Error", nil)
	}

	return utils.SuccessResponse(c, 200, "Comments retrieved successfully", result)
}

func (h *ShowcaseHandler) UpdateComment(c *fiber.Ctx) error {
	// Request ID for Tracing
	reqID, _ := c.Locals("requestid").(string)

	idStr := c.Params("id")
	commentID, err := uuid.Parse(idStr)
	if err != nil {
		h.log.WithFields(logrus.Fields{
			"request_id": reqID,
			"comment_id": idStr,
			"error":      err.Error(),
		}).Error("UpdateComment: Invalid Comment UUID")
		return utils.ErrorResponse(c, 400, "Invalid Comment UUID", nil)
	}

	userIDVal := c.Locals("user_id")
	if userIDVal == nil {
		h.log.WithFields(logrus.Fields{
			"request_id": reqID,
			"comment_id": commentID.String(),
		}).Error("UpdateComment: Unauthorized (User ID missing)")
		return utils.ErrorResponse(c, 401, "Unauthorized", nil)
	}
	var userID uuid.UUID
	if str, ok := userIDVal.(string); ok {
		userID, _ = uuid.Parse(str)
	} else if uid, ok := userIDVal.(uuid.UUID); ok {
		userID = uid
	} else {
		h.log.WithFields(logrus.Fields{
			"request_id": reqID,
			"comment_id": commentID.String(),
		}).Error("UpdateComment: Invalid User ID context")
		return utils.ErrorResponse(c, 401, "Invalid User ID context", nil)
	}

	var req entity.UpdateCommentRequest
	if err := c.BodyParser(&req); err != nil {
		h.log.WithFields(logrus.Fields{
			"request_id": reqID,
			"comment_id": commentID.String(),
			"error":      err.Error(),
		}).Error("UpdateComment: Invalid JSON body")
		return utils.ErrorResponse(c, 400, "Invalid JSON body", nil)
	}

	if req.Content == "" {
		h.log.WithFields(logrus.Fields{
			"request_id": reqID,
			"user_id":    userID.String(),
			"comment_id": commentID.String(),
		}).Error("UpdateComment: Content empty validation error")
		return utils.ErrorResponse(c, 400, "Content required", nil)
	}

	// LOG: Process Started
	h.log.WithFields(logrus.Fields{
		"request_id": reqID,
		"user_id":    userID.String(),
		"comment_id": commentID.String(),
	}).Info("Update comment process started")

	comment, err := h.usecase.UpdateComment(c.Context(), commentID, &req, userID)
	if err != nil {
		errMsg := err.Error()
		logFields := logrus.Fields{
			"request_id": reqID,
			"user_id":    userID.String(),
			"comment_id": commentID.String(),
			"error":      errMsg,
		}

		if strings.Contains(errMsg, "not found") {
			// LOG: ERROR - Resource Not Found
			h.log.WithFields(logFields).Error("UpdateComment: Comment not found")
			return utils.ErrorResponse(c, 404, errMsg, nil)
		}
		if strings.Contains(errMsg, "forbidden") {
			// LOG: WARN - Unauthorized Access
			h.log.WithFields(logFields).Warn("UpdateComment: Forbidden access (Not owner)")
			return utils.ErrorResponse(c, 403, errMsg, nil)
		}
		
		// LOG: ERROR - Database/Technical Error
		h.log.WithFields(logFields).Error("UpdateComment: Database or validation error")
		return utils.ErrorResponse(c, 500, "Internal Server Error", nil)
	}

	// LOG: Success
	h.log.WithFields(logrus.Fields{
		"request_id": reqID,
		"user_id":    userID.String(),
		"comment_id": commentID.String(),
	}).Info("Comment updated successfully")

	return utils.SuccessResponse(c, 200, "Comment updated successfully", fiber.Map{
		"comment": comment,
	})
}

func (h *ShowcaseHandler) DeleteComment(c *fiber.Ctx) error {
	// Request ID for Tracing
	reqID, _ := c.Locals("requestid").(string)

	idStr := c.Params("id")
	commentID, err := uuid.Parse(idStr)
	if err != nil {
		h.log.WithFields(logrus.Fields{
			"request_id": reqID,
			"comment_id": idStr,
			"error":      err.Error(),
		}).Error("DeleteComment: Invalid Comment UUID")
		return utils.ErrorResponse(c, 400, "Invalid Comment UUID", nil)
	}

	userIDVal := c.Locals("user_id")
	if userIDVal == nil {
		h.log.WithFields(logrus.Fields{
			"request_id": reqID,
			"comment_id": commentID.String(),
		}).Error("DeleteComment: Unauthorized (User ID missing)")
		return utils.ErrorResponse(c, 401, "Unauthorized", nil)
	}
	var userID uuid.UUID
	if str, ok := userIDVal.(string); ok {
		userID, _ = uuid.Parse(str)
	} else if uid, ok := userIDVal.(uuid.UUID); ok {
		userID = uid
	} else {
		h.log.WithFields(logrus.Fields{
			"request_id": reqID,
			"comment_id": commentID.String(),
		}).Error("DeleteComment: Invalid User ID context")
		return utils.ErrorResponse(c, 401, "Invalid User ID context", nil)
	}

	// LOG: Process Started
	h.log.WithFields(logrus.Fields{
		"request_id": reqID,
		"user_id":    userID.String(),
		"comment_id": commentID.String(),
	}).Info("Delete comment process started")

	if err := h.usecase.DeleteComment(c.Context(), commentID, userID); err != nil {
		errMsg := err.Error()
		logFields := logrus.Fields{
			"request_id": reqID,
			"user_id":    userID.String(),
			"comment_id": commentID.String(),
			"error":      errMsg,
		}

		if strings.Contains(errMsg, "not found") {
			// LOG: ERROR - Resource Not Found
			h.log.WithFields(logFields).Error("DeleteComment: Comment not found")
			return utils.ErrorResponse(c, 404, errMsg, nil)
		}
		if strings.Contains(errMsg, "forbidden") {
			// LOG: WARN - Unauthorized Access
			h.log.WithFields(logFields).Warn("DeleteComment: Forbidden access (Not owner)")
			return utils.ErrorResponse(c, 403, errMsg, nil)
		}
		
		// LOG: ERROR - Database/Technical Error
		h.log.WithFields(logFields).Error("DeleteComment: Database error")
		return utils.ErrorResponse(c, 500, "Internal Server Error", nil)
	}

	// LOG: Success
	h.log.WithFields(logrus.Fields{
		"request_id": reqID,
		"user_id":    userID.String(),
		"comment_id": commentID.String(),
	}).Info("Comment deleted/tombstoned successfully")

	return utils.SuccessResponse(c, 200, "Comment deleted successfully", nil)
}

// ToggleCommentLike toggles like/unlike on a comment
func (h *ShowcaseHandler) ToggleCommentLike(c *fiber.Ctx) error {
	// Request ID for Tracing
	reqID, _ := c.Locals("requestid").(string)

	idStr := c.Params("id")
	commentID, err := uuid.Parse(idStr)
	if err != nil {
		h.log.WithFields(logrus.Fields{
			"request_id": reqID,
			"comment_id": idStr,
			"error":      err.Error(),
		}).Error("ToggleCommentLike: Invalid Comment UUID")
		return utils.ErrorResponse(c, 400, "Invalid Comment UUID", nil)
	}

	userIDVal := c.Locals("user_id")
	if userIDVal == nil {
		h.log.WithFields(logrus.Fields{
			"request_id": reqID,
			"comment_id": commentID.String(),
		}).Error("ToggleCommentLike: Unauthorized (User ID missing)")
		return utils.ErrorResponse(c, 401, "Unauthorized", nil)
	}
	var userID uuid.UUID
	if str, ok := userIDVal.(string); ok {
		userID, _ = uuid.Parse(str)
	} else if uid, ok := userIDVal.(uuid.UUID); ok {
		userID = uid
	} else {
		h.log.WithFields(logrus.Fields{
			"request_id": reqID,
			"comment_id": commentID.String(),
		}).Error("ToggleCommentLike: Invalid User ID context")
		return utils.ErrorResponse(c, 401, "Invalid User ID context", nil)
	}

	// LOG: Process Started
	h.log.WithFields(logrus.Fields{
		"request_id": reqID,
		"user_id":    userID.String(),
		"comment_id": commentID.String(),
	}).Info("Toggle comment like process started")

	isLiked, likesCount, err := h.usecase.ToggleCommentLike(c.Context(), userID, commentID)
	if err != nil {
		errMsg := err.Error()
		logFields := logrus.Fields{
			"request_id": reqID,
			"user_id":    userID.String(),
			"comment_id": commentID.String(),
			"error":      errMsg,
		}

		if strings.Contains(errMsg, "not found") {
			// LOG: ERROR - Resource Not Found
			h.log.WithFields(logFields).Warn("ToggleCommentLike: Comment not found")
			return utils.ErrorResponse(c, 404, errMsg, nil)
		}
		
		// LOG: ERROR - Database/Technical Error
		h.log.WithFields(logFields).Error("ToggleCommentLike: Database error")
		return utils.ErrorResponse(c, 500, "Internal Server Error", nil)
	}

	status := "comment unliked"
	if isLiked {
		status = "comment liked"
	}

	// LOG: Success
	h.log.WithFields(logrus.Fields{
		"request_id": reqID,
		"user_id":    userID.String(),
		"comment_id": commentID.String(),
		"status":     status,
	}).Info("Toggle comment like action completed")

	responseData := map[string]interface{}{
		"liked":       isLiked,
		"likes_count": likesCount,
	}

	return utils.SuccessResponse(c, 200, "Success toggle like", responseData)
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

// RemoveCollaborator handles removing a collaborator (Kick or Leave)
func (h *ShowcaseHandler) RemoveCollaborator(c *fiber.Ctx) error {
	// Request ID for Tracing
	reqID, _ := c.Locals("requestid").(string)
	
	showcaseID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.ErrorResponse(c, 400, "Invalid showcase UUID", nil)
	}

	targetUserID, err := uuid.Parse(c.Params("userId"))
	if err != nil {
		h.log.WithFields(logrus.Fields{
			"request_id":   reqID,
			"target_id":    c.Params("userId"),
			"showcase_id":  showcaseID.String(),
			"error":        err.Error(),
		}).Error("RemoveCollaborator: Invalid Target UUID format")
		return utils.ErrorResponse(c, 400, "Invalid target user UUID", nil)
	}

	actorIDVal := c.Locals("user_id")
	if actorIDVal == nil {
		return utils.ErrorResponse(c, 401, "Unauthorized", nil)
	}
	var actorID uuid.UUID
	if str, ok := actorIDVal.(string); ok {
		actorID, _ = uuid.Parse(str)
	} else if uid, ok := actorIDVal.(uuid.UUID); ok {
		actorID = uid
	} else {
		return utils.ErrorResponse(c, 401, "Invalid User ID context", nil)
	}

	actionType := "kick"
	if actorID == targetUserID {
		actionType = "leave"
	}

	// LOG: Process Started
	h.log.WithFields(logrus.Fields{
		"request_id":   reqID,
		"user_id":      actorID.String(),
		"target_id":    targetUserID.String(),
		"showcase_id":  showcaseID.String(),
		"action_type":  actionType,
	}).Info("Remove collaborator process started")

	if err := h.usecase.RemoveCollaborator(c.Context(), showcaseID, targetUserID, actorID); err != nil {
		errMsg := err.Error()
		logFields := logrus.Fields{
			"request_id":   reqID,
			"user_id":      actorID.String(),
			"target_id":    targetUserID.String(),
			"showcase_id":  showcaseID.String(),
			"action_type":  actionType,
			"error":        errMsg,
		}

		if strings.Contains(errMsg, "not found") {
			// LOG: ERROR - Resource Not Found
			h.log.WithFields(logFields).Error("RemoveCollaborator: Resource not found")
			return utils.ErrorResponse(c, 404, errMsg, nil)
		}
		if strings.Contains(errMsg, "forbidden") {
			// LOG: WARN - Unauthorized Access
			h.log.WithFields(logFields).Warn("RemoveCollaborator: Forbidden access")
			return utils.ErrorResponse(c, 403, errMsg, nil)
		}
		if strings.Contains(errMsg, "target user is not a collaborator") || strings.Contains(errMsg, "owner cannot remove self") {
			// LOG: WARN - Logic Error (e.g. Owner self-remove check or target not member)
			h.log.WithFields(logFields).Warn("RemoveCollaborator: Invalid operation")
			return utils.ErrorResponse(c, 400, errMsg, nil)
		}
		
		// LOG: ERROR - Database/Technical Error
		h.log.WithFields(logFields).Error("RemoveCollaborator: Database or logic error")
		return utils.ErrorResponse(c, 500, "Internal Server Error", nil)
	}

	// LOG: Success
	h.log.WithFields(logrus.Fields{
		"request_id":   reqID,
		"user_id":      actorID.String(),
		"target_id":    targetUserID.String(),
		"showcase_id":  showcaseID.String(),
		"action_type":  actionType,
	}).Info("Collaborator removed/left successfully")

	return utils.SuccessResponse(c, 200, "Collaborator removed successfully", nil)
}

// GetCollaborators retrieves list of collaborators for a showcase
func (h *ShowcaseHandler) GetCollaborators(c *fiber.Ctx) error {
	showcaseID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.ErrorResponse(c, 400, "Invalid showcase UUID", nil)
	}

	actorIDVal := c.Locals("user_id")
	if actorIDVal == nil {
		return utils.ErrorResponse(c, 401, "Unauthorized", nil)
	}
	var actorID uuid.UUID
	if str, ok := actorIDVal.(string); ok {
		actorID, _ = uuid.Parse(str)
	} else if uid, ok := actorIDVal.(uuid.UUID); ok {
		actorID = uid
	} else {
		return utils.ErrorResponse(c, 401, "Invalid User ID context", nil)
	}

	collaborators, err := h.usecase.GetCollaborators(c.Context(), showcaseID, actorID)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "not found") {
			return utils.ErrorResponse(c, 404, errMsg, nil)
		}
		if strings.Contains(errMsg, "forbidden") {
			return utils.ErrorResponse(c, 403, errMsg, nil)
		}
		return utils.ErrorResponse(c, 500, "Internal Server Error", nil)
	}

	return utils.SuccessResponse(c, 200, "Collaborators retrieved successfully", map[string]interface{}{
		"collaborators": collaborators,
	})
}

// DeleteInvitation handles cancelling a pending invitation
func (h *ShowcaseHandler) DeleteInvitation(c *fiber.Ctx) error {
	// Request ID for Tracing
	reqID, _ := c.Locals("requestid").(string)
	
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		h.log.WithFields(logrus.Fields{
			"request_id":    reqID,
			"invitation_id": c.Params("id"),
			"error":         err.Error(),
		}).Error("DeleteInvitation: Invalid UUID format")
		return utils.ErrorResponse(c, 400, "Invalid invitation UUID", nil)
	}

	actorIDVal := c.Locals("user_id")
	if actorIDVal == nil {
		return utils.ErrorResponse(c, 401, "Unauthorized", nil)
	}
	var actorID uuid.UUID
	if str, ok := actorIDVal.(string); ok {
		actorID, _ = uuid.Parse(str)
	} else if uid, ok := actorIDVal.(uuid.UUID); ok {
		actorID = uid
	} else {
		return utils.ErrorResponse(c, 401, "Invalid User ID context", nil)
	}

	// LOG: Process Started
	h.log.WithFields(logrus.Fields{
		"request_id":    reqID,
		"user_id":       actorID.String(),
		"invitation_id": id.String(),
	}).Info("Delete invitation process started")

	if err := h.usecase.DeleteInvitation(c.Context(), id, actorID); err != nil {
		errMsg := err.Error()
		logFields := logrus.Fields{
			"request_id":    reqID,
			"user_id":       actorID.String(),
			"invitation_id": id.String(),
			"error":         errMsg,
		}

		if strings.Contains(errMsg, "not found") {
			// LOG: ERROR - Resource Not Found
			h.log.WithFields(logFields).Error("DeleteInvitation: Invitation not found")
			return utils.ErrorResponse(c, 404, errMsg, nil)
		}
		if strings.Contains(errMsg, "forbidden") {
			// LOG: WARN - Unauthorized Access
			h.log.WithFields(logFields).Warn("DeleteInvitation: Forbidden access")
			return utils.ErrorResponse(c, 403, errMsg, nil)
		}
		if strings.Contains(errMsg, "cannot delete processed invitation") {
			// LOG: WARN - Invalid State
			h.log.WithFields(logFields).Warn("DeleteInvitation: Invalid state (already processed)")
			return utils.ErrorResponse(c, 400, errMsg, nil)
		}
		
		// LOG: ERROR - Database/Technical Error
		h.log.WithFields(logFields).Error("DeleteInvitation: Database error")
		return utils.ErrorResponse(c, 500, "Internal Server Error", nil)
	}

	// LOG: Success
	h.log.WithFields(logrus.Fields{
		"request_id":    reqID,
		"user_id":       actorID.String(),
		"invitation_id": id.String(),
	}).Info("Invitation cancelled successfully")

	return utils.SuccessResponse(c, 200, "Invitation cancelled successfully", nil)
}

// GetPendingInvitations retrieves list of pending invitations for the current user
func (h *ShowcaseHandler) GetPendingInvitations(c *fiber.Ctx) error {
	actorIDVal := c.Locals("user_id")
	if actorIDVal == nil {
		return utils.ErrorResponse(c, 401, "Unauthorized", nil)
	}
	var actorID uuid.UUID
	if str, ok := actorIDVal.(string); ok {
		actorID, _ = uuid.Parse(str)
	} else if uid, ok := actorIDVal.(uuid.UUID); ok {
		actorID = uid
	} else {
		return utils.ErrorResponse(c, 401, "Invalid User ID context", nil)
	}

	limit := c.QueryInt("limit", 10)
	cursor := c.Query("cursor")

	invitations, meta, err := h.usecase.GetPendingInvitations(c.Context(), actorID, limit, cursor)
	if err != nil {
		return utils.ErrorResponse(c, 500, err.Error(), nil)
	}

	// Map to Response DTO
	var invitationResponses []map[string]interface{}
	for _, inv := range invitations {
		// Find Owner (Inviter)
		var inviter map[string]interface{}
		var showcaseInfo map[string]interface{}

		if inv.Showcase != nil {
			showcaseInfo = map[string]interface{}{
				"id":        inv.Showcase.ID,
				"title":     inv.Showcase.Title,
				"slug":      inv.Showcase.Slug,
				"cover_url": nil, // Map from Media if preloaded, but currently not preloaded in this flow specifically (only Showcase and Showcase.Collaborators). 
				// To get cover_url, we'd need Showcase.Media preloaded. 
				// Repository used Preload("Showcase") and Preload("Showcase.Collaborators"). Media is missing.
				// Leaving cover_url as nil or TODO.
			}

			// Find Owner
			for _, col := range inv.Showcase.Collaborators {
				if col.Role == entity.CollaborationRoleOwner {
					// Owner Found. Ideally we want Name/Username/Avatar.
					// But `col.User` might be nil if not preloaded.
					// Preload("Showcase.Collaborators") does NOT preload User inside those collaborators by default unless chained?
					// Usually GORM needs Preload("Showcase.Collaborators.User").
					// Repo didn't do that.
					// So we might only have Owner's UserID.
					// For now, returning minimal info or null if user info missing.
					// However, API Contract example shows username/avatar.
					// We'll set what we can (maybe just ID if that's all we have).
					inviter = map[string]interface{}{
						"user_id": col.UserID,
					}
					// If we had User info:
					// inviter["username"] = col.User.Username
					break
				}
			}
		}

		invitationResponses = append(invitationResponses, map[string]interface{}{
			"id":         inv.ID,
			"status":     inv.Status,
			"created_at": inv.CreatedAt,
			"expires_at": inv.ExpiredAt,
			"inviter":    inviter,
			"showcase":   showcaseInfo,
		})
	}

	return utils.SuccessResponse(c, 200, "Pending invitations retrieved", map[string]interface{}{
		"invitations": invitationResponses,
		"pagination":  meta,
	})
}

// InviteCollaborators handles sending invitations by usernames
func (h *ShowcaseHandler) InviteCollaborators(c *fiber.Ctx) error {
	// Request ID for Tracing
	reqID, _ := c.Locals("requestid").(string)
	
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.log.WithFields(logrus.Fields{
			"request_id":  reqID,
			"showcase_id": idStr,
			"error":       err.Error(),
		}).Error("InviteCollaborators: Invalid UUID format")
		return utils.ErrorResponse(c, 400, "Invalid showcase UUID", nil)
	}

	actorIDVal := c.Locals("user_id")
	if actorIDVal == nil {
		h.log.WithFields(logrus.Fields{
			"request_id":  reqID,
			"showcase_id": id.String(),
		}).Error("InviteCollaborators: Unauthorized (User ID missing)")
		return utils.ErrorResponse(c, 401, "Unauthorized", nil)
	}
	var actorID uuid.UUID
	if str, ok := actorIDVal.(string); ok {
		actorID, _ = uuid.Parse(str)
	} else if uid, ok := actorIDVal.(uuid.UUID); ok {
		actorID = uid
	} else {
		h.log.WithFields(logrus.Fields{
			"request_id":  reqID,
			"showcase_id": id.String(),
		}).Error("InviteCollaborators: Invalid User ID context")
		return utils.ErrorResponse(c, 401, "Invalid User ID context", nil)
	}

	var req struct {
		Usernames []string `json:"usernames" validate:"required,min=1,dive,required"`
	}

	if err := c.BodyParser(&req); err != nil {
		h.log.WithFields(logrus.Fields{
			"request_id":  reqID,
			"showcase_id": id.String(),
			"error":       err.Error(),
		}).Error("InviteCollaborators: Invalid request body")
		return utils.ErrorResponse(c, 400, "Invalid request body", nil)
	}

	// Validate (simple check)
	if len(req.Usernames) == 0 {
		return utils.ErrorResponse(c, 400, "Usernames cannot be empty", nil)
	}

	// LOG: Process Started
	h.log.WithFields(logrus.Fields{
		"request_id":  reqID,
		"inviter_id":  actorID.String(),
		"showcase_id": id.String(),
		"invitees":    req.Usernames,
	}).Info("Invite collaborator process started")

	invitedUsers, err := h.usecase.InviteCollaborators(c.Context(), id, req.Usernames, actorID)
	if err != nil {
		errMsg := err.Error()
		logFields := logrus.Fields{
			"request_id":  reqID,
			"inviter_id":  actorID.String(),
			"showcase_id": id.String(),
			"invitees":    req.Usernames,
			"error":       errMsg,
		}

		if strings.Contains(errMsg, "not found") {
			// LOG: ERROR - Resource Not Found (Showcase or Target User)
			h.log.WithFields(logFields).Error("InviteCollaborators: Resource not found")
			return utils.ErrorResponse(c, 404, errMsg, nil)
		}
		if strings.Contains(errMsg, "already") {
			// LOG: WARN - Duplicate Invitation
			h.log.WithFields(logFields).Warn("InviteCollaborators: Duplicate invitation or already collaborator")
			return utils.ErrorResponse(c, 400, errMsg, nil)
		}
		if strings.Contains(errMsg, "forbidden") {
			// LOG: WARN - Unauthorized Access
			h.log.WithFields(logFields).Warn("InviteCollaborators: Forbidden access (Not Owner)")
			return utils.ErrorResponse(c, 403, errMsg, nil)
		}
		if strings.Contains(errMsg, "self") { // Assuming usecase returns error containing "self" for self-invite
			// LOG: WARN - Self Invitation
			h.log.WithFields(logFields).Warn("InviteCollaborators: Self-invitation attempt")
			return utils.ErrorResponse(c, 400, errMsg, nil)
		}

		// LOG: ERROR - Database/Technical Error
		h.log.WithFields(logFields).Error("InviteCollaborators: Database or internal error")
		return utils.ErrorResponse(c, 500, "Internal Server Error", nil)
	}

	// LOG: Success
	h.log.WithFields(logrus.Fields{
		"request_id":  reqID,
		"inviter_id":  actorID.String(),
		"showcase_id": id.String(),
		"status":      "invitation_sent",
		"count":       len(invitedUsers),
	}).Info("Collaborator invitations sent successfully")

	return utils.SuccessResponse(c, 200, "Invitations sent successfully", map[string]interface{}{
		"message":       fmt.Sprintf("Undangan berhasil dikirim ke %d user", len(invitedUsers)),
		"invited_users": invitedUsers,
	})
}

// RespondInvitation handles accepting or rejecting an invitation
func (h *ShowcaseHandler) RespondInvitation(c *fiber.Ctx) error {
	// Request ID for Tracing
	reqID, _ := c.Locals("requestid").(string)
	
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		h.log.WithFields(logrus.Fields{
			"request_id":       reqID,
			"collaboration_id": c.Params("id"),
			"error":            err.Error(),
		}).Error("RespondInvitation: Invalid UUID format")
		return utils.ErrorResponse(c, 400, "Invalid invitation UUID", nil)
	}

	actorIDVal := c.Locals("user_id")
	if actorIDVal == nil {
		h.log.WithFields(logrus.Fields{
			"request_id":       reqID,
			"collaboration_id": id.String(),
		}).Error("RespondInvitation: Unauthorized (User ID missing)")
		return utils.ErrorResponse(c, 401, "Unauthorized", nil)
	}
	var actorID uuid.UUID
	if str, ok := actorIDVal.(string); ok {
		actorID, _ = uuid.Parse(str)
	} else if uid, ok := actorIDVal.(uuid.UUID); ok {
		actorID = uid
	} else {
		h.log.WithFields(logrus.Fields{
			"request_id":       reqID,
			"collaboration_id": id.String(),
		}).Error("RespondInvitation: Invalid User ID context")
		return utils.ErrorResponse(c, 401, "Invalid User ID context", nil)
	}

	var req struct {
		Response string `json:"response" validate:"required,oneof=ACCEPTED REJECTED"`
	}

	if err := c.BodyParser(&req); err != nil {
		h.log.WithFields(logrus.Fields{
			"request_id":       reqID,
			"collaboration_id": id.String(),
			"error":            err.Error(),
		}).Error("RespondInvitation: Invalid body")
		return utils.ErrorResponse(c, 400, "Invalid body", nil)
	}

	// Manual validation or rely on usecase
	if req.Response == "" {
		return utils.ErrorResponse(c, 400, "Response is required", nil)
	}

	// LOG: Process Started
	h.log.WithFields(logrus.Fields{
		"request_id":       reqID,
		"user_id":          actorID.String(),
		"collaboration_id": id.String(),
		"action":           req.Response,
	}).Info("Respond invitation process started")

	if err := h.usecase.RespondInvitation(c.Context(), id, actorID, req.Response); err != nil {
		errMsg := err.Error()
		logFields := logrus.Fields{
			"request_id":       reqID,
			"user_id":          actorID.String(),
			"collaboration_id": id.String(),
			"action":           req.Response,
			"error":            errMsg,
		}

		if strings.Contains(errMsg, "not found") {
			// LOG: ERROR - Resource Not Found
			h.log.WithFields(logFields).Error("RespondInvitation: Invitation not found")
			return utils.ErrorResponse(c, 404, errMsg, nil)
		}
		if strings.Contains(errMsg, "forbidden") {
			// LOG: WARN - Unauthorized Access
			h.log.WithFields(logFields).Warn("RespondInvitation: Forbidden access (Access denied)")
			return utils.ErrorResponse(c, 403, errMsg, nil)
		}
		if strings.Contains(errMsg, "no longer pending") || strings.Contains(errMsg, "invalid response") {
			// LOG: WARN - Invalid State
			h.log.WithFields(logFields).Warn("RespondInvitation: Invalid state or response")
			return utils.ErrorResponse(c, 400, errMsg, nil)
		}
		
		// LOG: ERROR - Database/Logic Error
		h.log.WithFields(logFields).Error("RespondInvitation: Database or logic error")
		return utils.ErrorResponse(c, 500, "Internal Server Error", nil)
	}

	msg := "Undangan berhasil diterima. Anda sekarang adalah collaborator."
	status := "invitation_accepted"
	if req.Response == "REJECTED" {
		msg = "Undangan berhasil ditolak."
		status = "invitation_rejected"
	}

	// LOG: Success
	h.log.WithFields(logrus.Fields{
		"request_id":       reqID,
		"user_id":          actorID.String(),
		"collaboration_id": id.String(),
		"status":           status,
	}).Info("Invitation response processed successfully")

	return utils.SuccessResponse(c, 200, msg, nil)
}

func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
