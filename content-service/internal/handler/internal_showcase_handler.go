package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/septianpadli/talas/content-service/internal/entity"
	"github.com/septianpadli/talas/content-service/internal/usecase"
	"github.com/sirupsen/logrus"
)

type InternalShowcaseHandler struct {
	showcaseUsecase usecase.ShowcaseUsecase
	log             *logrus.Logger
}

func NewInternalShowcaseHandler(
	showcaseUsecase usecase.ShowcaseUsecase,
	log *logrus.Logger,
) *InternalShowcaseHandler {
	return &InternalShowcaseHandler{
		showcaseUsecase: showcaseUsecase,
		log:             log,
	}
}

// GetShowcaseInternal returns lightweight showcase data for internal services
func (h *InternalShowcaseHandler) GetShowcaseInternal(c *fiber.Ctx) error {
	// Parse ID from params
	idParam := c.Params("id")
	showcaseID, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid showcase ID format",
		})
	}

	// Get showcase from usecase
	showcase, err := h.showcaseUsecase.GetShowcaseByID(c.Context(), showcaseID)
	if err != nil {
		h.log.WithError(err).Error("Failed to get showcase for internal API")
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Showcase not found",
		})
	}

	// Find owner from collaborators
	var ownerID uuid.UUID
	for _, collab := range showcase.Collaborators {
		if collab.Role == entity.CollaborationRoleOwner && collab.Status == entity.CollaborationStatusAccepted {
			ownerID = collab.UserID
			break
		}
	}

	// Return lightweight response
	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"id":       showcase.ID,
			"title":    showcase.Title,
			"slug":     showcase.Slug,
			"owner_id": ownerID,
		},
	})
}
