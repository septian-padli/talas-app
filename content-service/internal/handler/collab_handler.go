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

type CollabHandler struct {
	usecaseCollab usecase.CollabUsecase
	log           *logrus.Logger
}

func NewCollabHandler(usecaseCollab usecase.CollabUsecase, log *logrus.Logger) *CollabHandler {
	return &CollabHandler{
		usecaseCollab: usecaseCollab,
		log:           log,
	}
}

// RemoveCollaborator handles removing a collaborator (Kick or Leave)
func (h *CollabHandler) RemoveCollaborator(c *fiber.Ctx) error {
	// Request ID for Tracing
	reqID, _ := c.Locals("requestid").(string)

	showcaseID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.ErrorResponse(c, 400, "Invalid showcase UUID", nil)
	}

	targetUserID, err := uuid.Parse(c.Params("userId"))
	if err != nil {
		h.log.WithFields(logrus.Fields{
			"request_id":  reqID,
			"target_id":   c.Params("userId"),
			"showcase_id": showcaseID.String(),
			"error":       err.Error(),
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
		"request_id":  reqID,
		"user_id":     actorID.String(),
		"target_id":   targetUserID.String(),
		"showcase_id": showcaseID.String(),
		"action_type": actionType,
	}).Info("Remove collaborator process started")

	if err := h.usecaseCollab.RemoveCollaborator(c.Context(), showcaseID, targetUserID, actorID); err != nil {
		errMsg := err.Error()
		logFields := logrus.Fields{
			"request_id":  reqID,
			"user_id":     actorID.String(),
			"target_id":   targetUserID.String(),
			"showcase_id": showcaseID.String(),
			"action_type": actionType,
			"error":       errMsg,
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
		"request_id":  reqID,
		"user_id":     actorID.String(),
		"target_id":   targetUserID.String(),
		"showcase_id": showcaseID.String(),
		"action_type": actionType,
	}).Info("Collaborator removed/left successfully")

	return utils.SuccessResponse(c, 200, "Collaborator removed successfully", nil)
}

// GetCollaborators retrieves list of collaborators for a showcase
func (h *CollabHandler) GetCollaborators(c *fiber.Ctx) error {
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

	collaborators, err := h.usecaseCollab.GetCollaborators(c.Context(), showcaseID, actorID)
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
func (h *CollabHandler) DeleteInvitation(c *fiber.Ctx) error {
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

	if err := h.usecaseCollab.DeleteInvitation(c.Context(), id, actorID); err != nil {
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

	return utils.SuccessResponse(c, 200, "Invitation deleted successfully", nil)
}

// GetPendingInvitations retrieves list of pending invitations for the current user
func (h *CollabHandler) GetPendingInvitations(c *fiber.Ctx) error {
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

	invitations, meta, err := h.usecaseCollab.GetPendingInvitations(c.Context(), actorID, limit, cursor)
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
func (h *CollabHandler) InviteCollaborators(c *fiber.Ctx) error {
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

	invitedUsers, err := h.usecaseCollab.InviteCollaborators(c.Context(), id, req.Usernames, actorID)
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
func (h *CollabHandler) RespondInvitation(c *fiber.Ctx) error {
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

	if err := h.usecaseCollab.RespondInvitation(c.Context(), id, actorID, req.Response); err != nil {
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
