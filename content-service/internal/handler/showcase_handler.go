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


func (h *ShowcaseHandler) UpdateShowcase(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return utils.ErrorResponse(c, 400, "Invalid UUID format", nil)
	}

	var req entity.UpdateShowcaseRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, 400, "Invalid JSON body", nil)
	}

	// Parsing User ID from Locals
	userIDVal := c.Locals("user_id")
	if userIDVal == nil {
		return utils.ErrorResponse(c, 401, "Unauthorized", nil)
	}
	// Assuming middleware parsing logic or reusing Helper helper?
	// The Create method did: userIDStr, ok := c.Locals("user_id").(string) -> uuid.Parse.
	// GetMyShowcases did similar check.
	// I'll stick to string parsing for safety.
	var userID uuid.UUID
	if str, ok := userIDVal.(string); ok {
		userID, _ = uuid.Parse(str)
	} else if uid, ok := userIDVal.(uuid.UUID); ok {
		userID = uid
	} else {
		return utils.ErrorResponse(c, 401, "Invalid User ID context", nil)
	}

	result, err := h.usecase.UpdateShowcase(c.Context(), id, &req, userID)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "not found") {
			return utils.ErrorResponse(c, 404, "Showcase not found", nil)
		}
		if strings.Contains(errMsg, "forbidden") {
			return utils.ErrorResponse(c, 403, "Forbidden", nil)
		}
		if strings.Contains(errMsg, "content must be") || strings.Contains(errMsg, "invalid category") {
			return utils.ErrorResponse(c, 400, errMsg, nil)
		}
		return utils.ErrorResponse(c, 500, "Internal Server Error", nil)
	}

	return utils.SuccessResponse(c, 200, "Showcase updated successfully", result)
}


func (h *ShowcaseHandler) ToggleLike(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return utils.ErrorResponse(c, 400, "Invalid UUID format", nil)
	}

	userIDVal := c.Locals("user_id")
	if userIDVal == nil {
		return utils.ErrorResponse(c, 401, "Unauthorized", nil)
	}
	var userID uuid.UUID
	if str, ok := userIDVal.(string); ok {
		userID, _ = uuid.Parse(str)
	} else if uid, ok := userIDVal.(uuid.UUID); ok {
		userID = uid
	} else {
		return utils.ErrorResponse(c, 401, "Invalid User ID context", nil)
	}

	isLiked, err := h.usecase.ToggleLike(c.Context(), userID, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return utils.ErrorResponse(c, 404, "Showcase not found", nil)
		}
		return utils.ErrorResponse(c, 500, err.Error(), nil)
	}

	msg := "Showcase unliked"
	if isLiked {
		msg = "Showcase liked"
	}

	return utils.SuccessResponse(c, 200, msg, fiber.Map{"is_liked": isLiked})
}

func (h *ShowcaseHandler) ToggleBookmark(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return utils.ErrorResponse(c, 400, "Invalid UUID format", nil)
	}

	userIDVal := c.Locals("user_id")
	if userIDVal == nil {
		return utils.ErrorResponse(c, 401, "Unauthorized", nil)
	}
	var userID uuid.UUID
	if str, ok := userIDVal.(string); ok {
		userID, _ = uuid.Parse(str)
	} else if uid, ok := userIDVal.(uuid.UUID); ok {
		userID = uid
	} else {
		return utils.ErrorResponse(c, 401, "Invalid User ID context", nil)
	}

	isBookmarked, err := h.usecase.ToggleBookmark(c.Context(), userID, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return utils.ErrorResponse(c, 404, "Showcase not found", nil)
		}
		return utils.ErrorResponse(c, 500, err.Error(), nil)
	}

	msg := "Showcase unbookmarked"
	if isBookmarked {
		msg = "Showcase bookmarked"
	}

	return utils.SuccessResponse(c, 200, msg, fiber.Map{"is_bookmarked": isBookmarked})
}


func (h *ShowcaseHandler) CreateComment(c *fiber.Ctx) error {
	idStr := c.Params("id")
	showcaseID, err := uuid.Parse(idStr)
	if err != nil {
		return utils.ErrorResponse(c, 400, "Invalid Showcase UUID", nil)
	}

	userIDVal := c.Locals("user_id")
	if userIDVal == nil {
		return utils.ErrorResponse(c, 401, "Unauthorized", nil)
	}
	var userID uuid.UUID
	if str, ok := userIDVal.(string); ok {
		userID, _ = uuid.Parse(str)
	} else if uid, ok := userIDVal.(uuid.UUID); ok {
		userID = uid
	} else {
		return utils.ErrorResponse(c, 401, "Invalid User ID context", nil)
	}

	var req entity.CreateCommentRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, 400, "Invalid JSON body", nil)
	}

	if req.Content == "" {
		return utils.ErrorResponse(c, 400, "Content required", nil)
	}

	comment, err := h.usecase.CreateComment(c.Context(), showcaseID, &req, userID)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "showcase not found") || strings.Contains(errMsg, "parent comment not found") {
			return utils.ErrorResponse(c, 404, errMsg, nil)
		}
		if strings.Contains(errMsg, "content must be") || strings.Contains(errMsg, "invalid") || strings.Contains(errMsg, "belong") {
			return utils.ErrorResponse(c, 400, errMsg, nil)
		}
		return utils.ErrorResponse(c, 500, "Internal Server Error", nil)
	}

	return utils.SuccessResponse(c, 201, "Comment created successfully", fiber.Map{
		"comment": comment,
	})
}

func (h *ShowcaseHandler) ReplyComment(c *fiber.Ctx) error {
	idStr := c.Params("id")
	parentID, err := uuid.Parse(idStr)
	if err != nil {
		return utils.ErrorResponse(c, 400, "Invalid Parent Comment UUID", nil)
	}

	userIDVal := c.Locals("user_id")
	if userIDVal == nil {
		return utils.ErrorResponse(c, 401, "Unauthorized", nil)
	}
	var userID uuid.UUID
	if str, ok := userIDVal.(string); ok {
		userID, _ = uuid.Parse(str)
	} else if uid, ok := userIDVal.(uuid.UUID); ok {
		userID = uid
	} else {
		return utils.ErrorResponse(c, 401, "Invalid User ID context", nil)
	}

	var req entity.ReplyCommentRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, 400, "Invalid JSON body", nil)
	}

	if req.Content == "" {
		return utils.ErrorResponse(c, 400, "Content required", nil)
	}

	comment, err := h.usecase.ReplyComment(c.Context(), parentID, &req, userID)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "parent comment not found") {
			return utils.ErrorResponse(c, 404, errMsg, nil)
		}
		if strings.Contains(errMsg, "content must be") || strings.Contains(errMsg, "invalid") || strings.Contains(errMsg, "belong") {
			return utils.ErrorResponse(c, 400, errMsg, nil)
		}
		return utils.ErrorResponse(c, 500, "Internal Server Error", nil)
	}

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
	idStr := c.Params("id")
	commentID, err := uuid.Parse(idStr)
	if err != nil {
		return utils.ErrorResponse(c, 400, "Invalid Comment UUID", nil)
	}

	userIDVal := c.Locals("user_id")
	if userIDVal == nil {
		return utils.ErrorResponse(c, 401, "Unauthorized", nil)
	}
	var userID uuid.UUID
	if str, ok := userIDVal.(string); ok {
		userID, _ = uuid.Parse(str)
	} else if uid, ok := userIDVal.(uuid.UUID); ok {
		userID = uid
	} else {
		return utils.ErrorResponse(c, 401, "Invalid User ID context", nil)
	}

	var req entity.UpdateCommentRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, 400, "Invalid JSON body", nil)
	}

	if req.Content == "" {
		return utils.ErrorResponse(c, 400, "Content required", nil)
	}

	comment, err := h.usecase.UpdateComment(c.Context(), commentID, &req, userID)
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

	return utils.SuccessResponse(c, 200, "Comment updated successfully", fiber.Map{
		"comment": comment,
	})
}

func (h *ShowcaseHandler) DeleteComment(c *fiber.Ctx) error {
	idStr := c.Params("id")
	commentID, err := uuid.Parse(idStr)
	if err != nil {
		return utils.ErrorResponse(c, 400, "Invalid Comment UUID", nil)
	}

	userIDVal := c.Locals("user_id")
	if userIDVal == nil {
		return utils.ErrorResponse(c, 401, "Unauthorized", nil)
	}
	var userID uuid.UUID
	if str, ok := userIDVal.(string); ok {
		userID, _ = uuid.Parse(str)
	} else if uid, ok := userIDVal.(uuid.UUID); ok {
		userID = uid
	} else {
		return utils.ErrorResponse(c, 401, "Invalid User ID context", nil)
	}

	if err := h.usecase.DeleteComment(c.Context(), commentID, userID); err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "not found") {
			return utils.ErrorResponse(c, 404, errMsg, nil)
		}
		if strings.Contains(errMsg, "forbidden") {
			return utils.ErrorResponse(c, 403, errMsg, nil)
		}
		return utils.ErrorResponse(c, 500, "Internal Server Error", nil)
	}

	return utils.SuccessResponse(c, 200, "Comment deleted successfully", nil)
}

// ToggleCommentLike toggles like/unlike on a comment
func (h *ShowcaseHandler) ToggleCommentLike(c *fiber.Ctx) error {
	idStr := c.Params("id")
	commentID, err := uuid.Parse(idStr)
	if err != nil {
		return utils.ErrorResponse(c, 400, "Invalid Comment UUID", nil)
	}

	userIDVal := c.Locals("user_id")
	if userIDVal == nil {
		return utils.ErrorResponse(c, 401, "Unauthorized", nil)
	}
	var userID uuid.UUID
	if str, ok := userIDVal.(string); ok {
		userID, _ = uuid.Parse(str)
	} else if uid, ok := userIDVal.(uuid.UUID); ok {
		userID = uid
	} else {
		return utils.ErrorResponse(c, 401, "Invalid User ID context", nil)
	}

	isLiked, likesCount, err := h.usecase.ToggleCommentLike(c.Context(), userID, commentID)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "not found") {
			return utils.ErrorResponse(c, 404, errMsg, nil)
		}
		return utils.ErrorResponse(c, 500, "Internal Server Error", nil)
	}

	responseData := map[string]interface{}{
		"liked":       isLiked,
		"likes_count": likesCount,
	}

	return utils.SuccessResponse(c, 200, "Success toggle like", responseData)
}

// DeleteShowcase handles deleting a showcase
func (h *ShowcaseHandler) DeleteShowcase(c *fiber.Ctx) error {
	showcaseID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.ErrorResponse(c, 400, "Invalid showcase UUID", nil)
	}

	userIDVal := c.Locals("user_id")
	if userIDVal == nil {
		return utils.ErrorResponse(c, 401, "Unauthorized", nil)
	}
	var userID uuid.UUID
	if str, ok := userIDVal.(string); ok {
		userID, _ = uuid.Parse(str)
	} else if uid, ok := userIDVal.(uuid.UUID); ok {
		userID = uid
	} else {
		return utils.ErrorResponse(c, 401, "Invalid User ID context", nil)
	}

	if err := h.usecase.DeleteShowcase(c.Context(), showcaseID, userID); err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "not found") {
			return utils.ErrorResponse(c, 404, errMsg, nil)
		}
		if strings.Contains(errMsg, "forbidden") {
			return utils.ErrorResponse(c, 403, errMsg, nil)
		}
		return utils.ErrorResponse(c, 500, "Internal Server Error", nil)
	}

	return utils.SuccessResponse(c, 200, "Showcase successfully deleted (archived)", nil)
}

// RemoveCollaborator handles removing a collaborator (Kick or Leave)
func (h *ShowcaseHandler) RemoveCollaborator(c *fiber.Ctx) error {
	showcaseID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.ErrorResponse(c, 400, "Invalid showcase UUID", nil)
	}

	targetUserID, err := uuid.Parse(c.Params("userId"))
	if err != nil {
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

	if err := h.usecase.RemoveCollaborator(c.Context(), showcaseID, targetUserID, actorID); err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "not found") {
			return utils.ErrorResponse(c, 404, errMsg, nil)
		}
		if strings.Contains(errMsg, "forbidden") {
			return utils.ErrorResponse(c, 403, errMsg, nil)
		}
		if strings.Contains(errMsg, "target user is not a collaborator") {
			return utils.ErrorResponse(c, 400, errMsg, nil)
		}
		return utils.ErrorResponse(c, 500, "Internal Server Error", nil)
	}

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
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
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

	if err := h.usecase.DeleteInvitation(c.Context(), id, actorID); err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "not found") {
			return utils.ErrorResponse(c, 404, errMsg, nil)
		}
		if strings.Contains(errMsg, "forbidden") {
			return utils.ErrorResponse(c, 403, errMsg, nil)
		}
		if strings.Contains(errMsg, "cannot delete processed invitation") {
			return utils.ErrorResponse(c, 400, errMsg, nil)
		}
		return utils.ErrorResponse(c, 500, "Internal Server Error", nil)
	}

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
	id, err := uuid.Parse(c.Params("id"))
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

	var req struct {
		Usernames []string `json:"usernames" validate:"required,min=1,dive,required"`
	}

	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, 400, "Invalid request body", nil)
	}

	// Validate (simple check)
	if len(req.Usernames) == 0 {
		return utils.ErrorResponse(c, 400, "Usernames cannot be empty", nil)
	}

	invitedUsers, err := h.usecase.InviteCollaborators(c.Context(), id, req.Usernames, actorID)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "already") {
			return utils.ErrorResponse(c, 400, errMsg, nil)
		}
		if strings.Contains(errMsg, "forbidden") {
			return utils.ErrorResponse(c, 403, errMsg, nil)
		}
		return utils.ErrorResponse(c, 500, "Internal Server Error", nil)
	}

	return utils.SuccessResponse(c, 200, "Invitations sent successfully", map[string]interface{}{
		"message":       fmt.Sprintf("Undangan berhasil dikirim ke %d user", len(invitedUsers)),
		"invited_users": invitedUsers,
	})
}

// RespondInvitation handles accepting or rejecting an invitation
func (h *ShowcaseHandler) RespondInvitation(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
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

	var req struct {
		Response string `json:"response" validate:"required,oneof=ACCEPTED REJECTED"`
	}

	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, 400, "Invalid body", nil)
	}

	// Manual validation or rely on usecase
	if req.Response == "" {
		return utils.ErrorResponse(c, 400, "Response is required", nil)
	}

	if err := h.usecase.RespondInvitation(c.Context(), id, actorID, req.Response); err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "not found") {
			return utils.ErrorResponse(c, 404, errMsg, nil)
		}
		if strings.Contains(errMsg, "forbidden") {
			return utils.ErrorResponse(c, 403, errMsg, nil)
		}
		if strings.Contains(errMsg, "no longer pending") || strings.Contains(errMsg, "invalid response") {
			return utils.ErrorResponse(c, 400, errMsg, nil)
		}
		return utils.ErrorResponse(c, 500, "Internal Server Error", nil)
	}

	msg := "Undangan berhasil diterima. Anda sekarang adalah collaborator."
	if req.Response == "REJECTED" {
		msg = "Undangan berhasil ditolak."
	}

	return utils.SuccessResponse(c, 200, msg, nil)
}

func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
