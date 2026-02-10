package handler

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/septianpadli/talas/content-service/internal/entity"
	"github.com/septianpadli/talas/content-service/internal/usecase"
	"github.com/septianpadli/talas/content-service/pkg/utils"
	"github.com/sirupsen/logrus"
)

type CommentHandler struct {
	usecase usecase.CommentUsecase
	log     *logrus.Logger
}

func NewCommentHandler(usecase usecase.CommentUsecase, log *logrus.Logger) *CommentHandler {
	return &CommentHandler{
		usecase: usecase,
		log:     log,
	}
}

func (h *CommentHandler) CreateComment(c *fiber.Ctx) error {
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

func (h *CommentHandler) ReplyComment(c *fiber.Ctx) error {
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

func (h *CommentHandler) GetShowcaseComments(c *fiber.Ctx) error {
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

func (h *CommentHandler) UpdateComment(c *fiber.Ctx) error {
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

func (h *CommentHandler) DeleteComment(c *fiber.Ctx) error {
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
func (h *CommentHandler) ToggleCommentLike(c *fiber.Ctx) error {
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
