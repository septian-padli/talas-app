package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/septianpadli/talas/content-service/internal/usecase"
	"github.com/septianpadli/talas/content-service/pkg/utils"
)

type ProjectHandler struct {
	usecase usecase.ProjectUsecase
}

func NewProjectHandler(usecase usecase.ProjectUsecase) *ProjectHandler {
	return &ProjectHandler{usecase: usecase}
}

func (h *ProjectHandler) CreateProject(c *fiber.Ctx) error {
	// Logic placeholder
	return utils.SuccessResponse(c, 201, "Project created successfully", nil)
}
