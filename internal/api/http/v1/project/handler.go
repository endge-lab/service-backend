package project

import (
	transport "github.com/endge-lab/service-backend/internal/api/http/v1/transport"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase"
	"github.com/endge-lab/service-backend/internal/usecase/adapters"
	servicefiber "github.com/endge-lab/service-kit-go/pkg/httpkit/fiber"
	"github.com/endge-lab/service-kit-go/pkg/logging"
	appvalidator "github.com/endge-lab/service-kit-go/pkg/validator"
	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type ErrorResponse = transport.ErrorResponse

type Handler struct {
	projectService adapters.ProjectService
	validator      appvalidator.Validator
	logger         *zap.Logger
	tracer         trace.Tracer
}

func NewHandler(
	service *usecase.Service,
	validator appvalidator.Validator,
	logger *zap.Logger,
	tracer trace.Tracer,
) *Handler {
	return &Handler{
		projectService: service.Projects,
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
// @Failure 400 {object} transport.ErrorResponse
// @Failure 409 {object} transport.ErrorResponse
// @Failure 500 {object} transport.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects [post]
func (h *Handler) CreateProject(c *fiber.Ctx) error {
	logger := logging.WithContext(c.UserContext(), h.logger).With(zap.String("handler", "create_project"))

	var request CreateProjectRequest
	if err := c.BodyParser(&request); err != nil {
		return transport.WriteErrorResponse(c, transport.ErrInvalidBody)
	}
	if err := h.validator.Validate(&request); err != nil {
		return transport.WriteErrorResponse(c, transport.ErrValidationError)
	}

	project, err := h.projectService.Create(c.UserContext(), adapters.CreateProjectInput{
		Identity:    request.Identity,
		DisplayName: request.DisplayName,
		Description: request.Description,
		Active:      request.Active,
		Meta:        request.Meta,
	})
	if err != nil {
		return h.respondDomainError(c, err)
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
// @Failure 500 {object} transport.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects [get]
func (h *Handler) ListProjects(c *fiber.Ctx) error {
	projects, err := h.projectService.List(c.UserContext())
	if err != nil {
		return h.respondDomainError(c, err)
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
// @Failure 400 {object} transport.ErrorResponse
// @Failure 404 {object} transport.ErrorResponse
// @Failure 500 {object} transport.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{project_identity} [get]
func (h *Handler) GetProjectByIdentity(c *fiber.Ctx) error {
	project, err := h.projectService.GetByIdentity(c.UserContext(), c.Params("project_identity"))
	if err != nil {
		return h.respondDomainError(c, err)
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
// @Failure 400 {object} transport.ErrorResponse
// @Failure 404 {object} transport.ErrorResponse
// @Failure 500 {object} transport.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{project_identity} [patch]
func (h *Handler) UpdateProject(c *fiber.Ctx) error {
	var request UpdateProjectRequest
	if err := c.BodyParser(&request); err != nil {
		return transport.WriteErrorResponse(c, transport.ErrInvalidBody)
	}
	if err := h.validator.Validate(&request); err != nil {
		return transport.WriteErrorResponse(c, transport.ErrValidationError)
	}

	project, err := h.projectService.Update(c.UserContext(), adapters.UpdateProjectInput{
		Identity:    c.Params("project_identity"),
		DisplayName: request.DisplayName,
		Description: request.Description,
		Active:      request.Active,
		Meta:        request.Meta,
	})
	if err != nil {
		return h.respondDomainError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(NewProjectResponse(project))
}

// SoftDeleteProject godoc
// @Summary Удалить проект
// @Description Выполняет soft delete проекта по identity.
// @Tags projects
// @Param project_identity path string true "Project identity" example(demo-project)
// @Success 204
// @Failure 400 {object} transport.ErrorResponse
// @Failure 404 {object} transport.ErrorResponse
// @Failure 500 {object} transport.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{project_identity} [delete]
func (h *Handler) SoftDeleteProject(c *fiber.Ctx) error {
	if err := h.projectService.SoftDelete(c.UserContext(), c.Params("project_identity")); err != nil {
		return h.respondDomainError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// RestoreProject godoc
// @Summary Восстановить проект
// @Description Восстанавливает soft-deleted проект по identity.
// @Tags projects
// @Param project_identity path string true "Project identity" example(demo-project)
// @Success 204
// @Failure 400 {object} transport.ErrorResponse
// @Failure 404 {object} transport.ErrorResponse
// @Failure 500 {object} transport.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{project_identity}/restore [post]
func (h *Handler) RestoreProject(c *fiber.Ctx) error {
	if err := h.projectService.Restore(c.UserContext(), c.Params("project_identity")); err != nil {
		return h.respondDomainError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// HardDeleteProject godoc
// @Summary Удалить проект физически
// @Description Выполняет hard delete проекта по identity.
// @Tags projects
// @Param project_identity path string true "Project identity" example(demo-project)
// @Success 204
// @Failure 400 {object} transport.ErrorResponse
// @Failure 404 {object} transport.ErrorResponse
// @Failure 500 {object} transport.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{project_identity}/hard [delete]
func (h *Handler) HardDeleteProject(c *fiber.Ctx) error {
	if err := h.projectService.HardDelete(c.UserContext(), c.Params("project_identity")); err != nil {
		return h.respondDomainError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) TraceMiddleware(spanName string) fiber.Handler {
	return servicefiber.TraceMiddleware(h.tracer, h.logger, spanName)
}

func (h *Handler) respondDomainError(c *fiber.Ctx, err error) error {
	fields := []zap.Field{
		zap.Error(err),
		zap.String("error_code", domainerrors.CodeOf(err)),
		zap.String("method", c.Method()),
		zap.String("path", c.Path()),
	}

	logger := logging.WithContext(c.UserContext(), h.logger)
	if domainerrors.HTTPStatusOf(err) >= fiber.StatusInternalServerError {
		logger.Error("unexpected request transport", fields...)
	} else {
		logger.Warn("request completed with business transport", fields...)
	}

	return transport.WriteErrorResponse(c, err)
}
