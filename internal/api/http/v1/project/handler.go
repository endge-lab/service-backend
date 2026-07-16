package project

import (
	respond "github.com/endge-lab/service-backend/internal/api/http/respond"
	"github.com/endge-lab/service-backend/internal/usecase/projects"
	"github.com/endge-lab/service-kit-go/pkg/logging"
	appvalidator "github.com/endge-lab/service-kit-go/pkg/validator"
	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type ErrorResponse = respond.ErrorResponse

type Handler struct {
	projectService UseCase
	validator      appvalidator.Validator
	logger         *zap.Logger
	tracer         trace.Tracer
}

func NewHandler(
	service UseCase,
	validator appvalidator.Validator,
	logger *zap.Logger,
	tracer trace.Tracer,
) *Handler {
	return &Handler{
		projectService: service,
		validator:      validator,
		logger:         logger.With(zap.String("component", "project_http_handler")),
		tracer:         tracer,
	}
}

// CreateProject godoc
// @Summary Создать проект
// @Description Создает новый проект.
// @Tags projects
// @Accept json
// @Produce json
// @Param request body CreateProjectRequest true "Параметры проекта"
// @Success 201 {object} ProjectResponse
// @Failure 400 {object} respond.ErrorResponse
// @Failure 409 {object} respond.ErrorResponse
// @Failure 500 {object} respond.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects [post]
func (h *Handler) CreateProject(c *fiber.Ctx) error {
	logger := logging.WithContext(c.UserContext(), h.logger).With(zap.String("handler", "create_project"))

	var request CreateProjectRequest
	if err := c.BodyParser(&request); err != nil {
		return respond.WriteErrorResponse(c, respond.ErrInvalidBody)
	}
	if err := h.validator.Validate(&request); err != nil {
		return respond.WriteErrorResponse(c, respond.ErrValidationError)
	}

	project, err := h.projectService.Create(c.UserContext(), projects.CreateProjectInput{
		Identity:    request.Identity,
		DisplayName: request.DisplayName,
		Description: request.Description,
		Active:      request.Active,
		Meta:        request.Meta,
	})
	if err != nil {
		return respond.RespondDomainError(c, h.logger, err)
	}

	logger.Debug("create project handler completed", zap.String("project_id", project.ID.String()))
	return c.Status(fiber.StatusCreated).JSON(NewProjectResponse(project))
}

// ListProjects godoc
// @Summary Список проектов
// @Description Возвращает список неудаленных проектов.
// @Tags projects
// @Produce json
// @Success 200 {object} ProjectsListResponse
// @Failure 500 {object} respond.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects [get]
func (h *Handler) ListProjects(c *fiber.Ctx) error {
	projects, err := h.projectService.List(c.UserContext())
	if err != nil {
		return respond.RespondDomainError(c, h.logger, err)
	}

	return c.Status(fiber.StatusOK).JSON(NewProjectsListResponse(projects))
}

// GetProjectByIdentity godoc
// @Summary Получить проект
// @Description Возвращает проект по identity.
// @Tags projects
// @Produce json
// @Param project_identity path string true "Project identity" example(demo-project)
// @Success 200 {object} ProjectResponse
// @Failure 400 {object} respond.ErrorResponse
// @Failure 404 {object} respond.ErrorResponse
// @Failure 500 {object} respond.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{project_identity} [get]
func (h *Handler) GetProjectByIdentity(c *fiber.Ctx) error {
	project, err := h.projectService.GetByIdentity(c.UserContext(), c.Params("project_identity"))
	if err != nil {
		return respond.RespondDomainError(c, h.logger, err)
	}

	return c.Status(fiber.StatusOK).JSON(NewProjectResponse(project))
}

// UpdateProject godoc
// @Summary Обновить проект
// @Description Обновляет проект по identity.
// @Tags projects
// @Accept json
// @Produce json
// @Param project_identity path string true "Project identity" example(demo-project)
// @Param request body UpdateProjectRequest true "Параметры обновления проекта"
// @Success 200 {object} ProjectResponse
// @Failure 400 {object} respond.ErrorResponse
// @Failure 404 {object} respond.ErrorResponse
// @Failure 500 {object} respond.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{project_identity} [patch]
func (h *Handler) UpdateProject(c *fiber.Ctx) error {
	var request UpdateProjectRequest
	if err := c.BodyParser(&request); err != nil {
		return respond.WriteErrorResponse(c, respond.ErrInvalidBody)
	}
	if err := h.validator.Validate(&request); err != nil {
		return respond.WriteErrorResponse(c, respond.ErrValidationError)
	}

	project, err := h.projectService.Update(c.UserContext(), projects.UpdateProjectInput{
		Identity:    c.Params("project_identity"),
		DisplayName: request.DisplayName,
		Description: request.Description,
		Active:      request.Active,
		Meta:        request.Meta,
	})
	if err != nil {
		return respond.RespondDomainError(c, h.logger, err)
	}

	return c.Status(fiber.StatusOK).JSON(NewProjectResponse(project))
}

// SoftDeleteProject godoc
// @Summary Удалить проект
// @Description Выполняет soft delete проекта по identity.
// @Tags projects
// @Param project_identity path string true "Project identity" example(demo-project)
// @Success 204
// @Failure 400 {object} respond.ErrorResponse
// @Failure 404 {object} respond.ErrorResponse
// @Failure 500 {object} respond.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{project_identity} [delete]
func (h *Handler) SoftDeleteProject(c *fiber.Ctx) error {
	if err := h.projectService.SoftDelete(c.UserContext(), c.Params("project_identity")); err != nil {
		return respond.RespondDomainError(c, h.logger, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// RestoreProject godoc
// @Summary Восстановить проект
// @Description Восстанавливает soft-deleted проект по identity.
// @Tags projects
// @Param project_identity path string true "Project identity" example(demo-project)
// @Success 204
// @Failure 400 {object} respond.ErrorResponse
// @Failure 404 {object} respond.ErrorResponse
// @Failure 500 {object} respond.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{project_identity}/restore [post]
func (h *Handler) RestoreProject(c *fiber.Ctx) error {
	if err := h.projectService.Restore(c.UserContext(), c.Params("project_identity")); err != nil {
		return respond.RespondDomainError(c, h.logger, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// HardDeleteProject godoc
// @Summary Удалить проект физически
// @Description Выполняет hard delete проекта по identity.
// @Tags projects
// @Param project_identity path string true "Project identity" example(demo-project)
// @Success 204
// @Failure 400 {object} respond.ErrorResponse
// @Failure 404 {object} respond.ErrorResponse
// @Failure 500 {object} respond.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{project_identity}/hard [delete]
func (h *Handler) HardDeleteProject(c *fiber.Ctx) error {
	if err := h.projectService.HardDelete(c.UserContext(), c.Params("project_identity")); err != nil {
		return respond.RespondDomainError(c, h.logger, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}
