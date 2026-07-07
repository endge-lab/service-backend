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
	"github.com/google/uuid"
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
// @Router /api/projects [post]
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
		Identity:              request.Identity,
		DisplayName:           request.DisplayName,
		ExtendSettings:        request.ExtendSettings,
		SettingsID:            request.SettingsID,
		NavigationID:          request.NavigationID,
		FolderID:              request.FolderID,
		AllowedEnvironmentIDs: request.AllowedEnvironmentIDs,
		Meta:                  request.Meta,
	})
	if err != nil {
		return h.respondDomainError(c, err)
	}

	logger.Debug("create project handler completed", zap.String("project_id", project.ID.String()))
	return c.Status(fiber.StatusCreated).JSON(NewProjectResponse(project))
}

// GetProjectByID godoc
// @Summary Получить проект по ID
// @Description Возвращает проект по UUID.
// @Tags projects
// @Accept json
// @Produce json
// @Param id path string true "Project ID" example(00000000-0000-4000-8000-000000000001)
// @Success 200 {object} ProjectResponse
// @Failure 400 {object} transport.ErrorResponse
// @Failure 404 {object} transport.ErrorResponse
// @Failure 500 {object} transport.ErrorResponse
// @Security BearerAuth
// @Router /api/projects/{id} [get]
func (h *Handler) GetProjectByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return transport.WriteErrorResponse(c, transport.ErrValidationError)
	}

	project, err := h.projectService.GetByID(c.UserContext(), id)
	if err != nil {
		return h.respondDomainError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(NewProjectResponse(project))
}

// GetProjectByIdentity godoc
// @Summary Получить проект по identity
// @Description Возвращает проект по уникальному identity.
// @Tags projects
// @Accept json
// @Produce json
// @Param identity path string true "Project identity" example(swagger-project)
// @Success 200 {object} ProjectResponse
// @Failure 400 {object} transport.ErrorResponse
// @Failure 404 {object} transport.ErrorResponse
// @Failure 500 {object} transport.ErrorResponse
// @Security BearerAuth
// @Router /api/projects/identity/{identity} [get]
func (h *Handler) GetProjectByIdentity(c *fiber.Ctx) error {
	identity := c.Params("identity")

	project, err := h.projectService.GetByIdentity(c.UserContext(), identity)
	if err != nil {
		return h.respondDomainError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(NewProjectResponse(project))
}

// ListProjects godoc
// @Summary Список проектов
// @Description Возвращает список неудаленных проектов.
// @Tags projects
// @Accept json
// @Produce json
// @Success 200 {object} ProjectsListResponse
// @Failure 500 {object} transport.ErrorResponse
// @Security BearerAuth
// @Router /api/projects [get]
func (h *Handler) ListProjects(c *fiber.Ctx) error {
	projects, err := h.projectService.List(c.UserContext())
	if err != nil {
		return h.respondDomainError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(NewProjectsListResponse(projects))
}

// UpdateProject godoc
// @Summary Обновить проект
// @Description Обновляет проект по UUID.
// @Tags projects
// @Accept json
// @Produce json
// @Param id path string true "Project ID" example(00000000-0000-4000-8000-000000000001)
// @Param request body UpdateProjectRequest true "Параметры обновления проекта"
// @Success 200 {object} ProjectResponse
// @Failure 400 {object} transport.ErrorResponse
// @Failure 404 {object} transport.ErrorResponse
// @Failure 409 {object} transport.ErrorResponse
// @Failure 500 {object} transport.ErrorResponse
// @Security BearerAuth
// @Router /api/projects/{id} [patch]
func (h *Handler) UpdateProject(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return transport.WriteErrorResponse(c, transport.ErrValidationError)
	}

	var request UpdateProjectRequest
	if err := c.BodyParser(&request); err != nil {
		return transport.WriteErrorResponse(c, transport.ErrInvalidBody)
	}

	if err := h.validator.Validate(&request); err != nil {
		return transport.WriteErrorResponse(c, transport.ErrValidationError)
	}

	project, err := h.projectService.Update(c.UserContext(), adapters.UpdateProjectInput{
		ID:                    id,
		Identity:              request.Identity,
		DisplayName:           request.DisplayName,
		ExtendSettings:        request.ExtendSettings,
		SettingsID:            request.SettingsID,
		NavigationID:          request.NavigationID,
		FolderID:              request.FolderID,
		AllowedEnvironmentIDs: request.AllowedEnvironmentIDs,
		Meta:                  request.Meta,
	})
	if err != nil {
		return h.respondDomainError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(NewProjectResponse(project))
}

// SoftDeleteProject godoc
// @Summary Удалить проект
// @Description Выполняет soft delete проекта.
// @Tags projects
// @Accept json
// @Produce json
// @Param id path string true "Project ID" example(00000000-0000-4000-8000-000000000001)
// @Success 204
// @Failure 400 {object} transport.ErrorResponse
// @Failure 404 {object} transport.ErrorResponse
// @Failure 500 {object} transport.ErrorResponse
// @Security BearerAuth
// @Router /api/projects/{id} [delete]
func (h *Handler) SoftDeleteProject(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return transport.WriteErrorResponse(c, transport.ErrValidationError)
	}

	if err := h.projectService.SoftDelete(c.UserContext(), id); err != nil {
		return h.respondDomainError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// RestoreProject godoc
// @Summary Восстановить проект
// @Description Восстанавливает soft-deleted проект.
// @Tags projects
// @Accept json
// @Produce json
// @Param id path string true "Project ID" example(00000000-0000-4000-8000-000000000001)
// @Success 204
// @Failure 400 {object} transport.ErrorResponse
// @Failure 404 {object} transport.ErrorResponse
// @Failure 500 {object} transport.ErrorResponse
// @Security BearerAuth
// @Router /api/projects/{id}/restore [post]
func (h *Handler) RestoreProject(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return transport.WriteErrorResponse(c, transport.ErrValidationError)
	}

	if err := h.projectService.Restore(c.UserContext(), id); err != nil {
		return h.respondDomainError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// HardDeleteProject godoc
// @Summary Удалить проект физически
// @Description Выполняет hard delete проекта из базы.
// @Tags projects
// @Accept json
// @Produce json
// @Param id path string true "Project ID" example(00000000-0000-4000-8000-000000000001)
// @Success 204
// @Failure 400 {object} transport.ErrorResponse
// @Failure 404 {object} transport.ErrorResponse
// @Failure 500 {object} transport.ErrorResponse
// @Security BearerAuth
// @Router /api/projects/{id}/hard [delete]
func (h *Handler) HardDeleteProject(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return transport.WriteErrorResponse(c, transport.ErrValidationError)
	}

	if err := h.projectService.HardDelete(c.UserContext(), id); err != nil {
		return h.respondDomainError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// CountProjects godoc
// @Summary Количество проектов
// @Description Возвращает количество неудаленных проектов.
// @Tags projects
// @Accept json
// @Produce json
// @Success 200 {object} CountProjectsResponse
// @Failure 500 {object} transport.ErrorResponse
// @Security BearerAuth
// @Router /api/projects/count [get]
func (h *Handler) CountProjects(c *fiber.Ctx) error {
	count, err := h.projectService.Count(c.UserContext())
	if err != nil {
		return h.respondDomainError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(CountProjectsResponse{Count: count})
}

func (h *Handler) TraceMiddleware(spanName string) fiber.Handler {
	return servicefiber.TraceMiddleware(h.tracer, h.logger, spanName)
}

func (h *Handler) respondDomainError(c *fiber.Ctx, err error) error {
	return h.respondUnexpectedError(c, err)
}

func (h *Handler) respondUnexpectedError(c *fiber.Ctx, err error) error {
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
