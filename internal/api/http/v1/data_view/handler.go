package data_view

import (
	transport "github.com/endge-lab/service-backend/internal/api/http/v1/transport"
	"github.com/endge-lab/service-backend/internal/usecase"
	"github.com/endge-lab/service-backend/internal/usecase/adapters"
	appvalidator "github.com/endge-lab/service-kit-go/pkg/validator"
	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type Handler struct {
	service   adapters.DataViewService
	validator appvalidator.Validator
	logger    *zap.Logger
	tracer    trace.Tracer
}

func NewHandler(s *usecase.Service, v appvalidator.Validator, l *zap.Logger, t trace.Tracer) *Handler {
	return &Handler{service: s.DataViews, validator: v, logger: l.With(zap.String("component", "data_view_http_handler")), tracer: t}
}

// Create godoc
// @Summary Создать DataView
// @Description Создает DataView в папке проекта и связывает его с активной Query этого же проекта. Source и schema хранятся как authoring configuration и не компилируются и не выполняются.
// @Tags data-views
// @Accept json
// @Produce json
// @Param project_identity path string true "Project identity" example(demo-project)
// @Param request body CreateDataViewRequest true "Параметры DataView"
// @Success 201 {object} DataViewResponse
// @Failure 400 {object} transport.ErrorResponse
// @Failure 404 {object} transport.ErrorResponse
// @Failure 409 {object} transport.ErrorResponse
// @Failure 500 {object} transport.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{project_identity}/data-views [post]
func (h *Handler) Create(c *fiber.Ctx) error {
	var request CreateDataViewRequest
	if err := c.BodyParser(&request); err != nil {
		return transport.WriteErrorResponse(c, transport.ErrInvalidBody)
	}
	if err := h.validator.Validate(&request); err != nil {
		return transport.WriteErrorResponse(c, transport.ErrValidationError)
	}
	project := c.Params("project_identity")
	value, err := h.service.Create(c.UserContext(), adapters.CreateDataViewInput{ProjectIdentity: project, FolderIdentity: request.FolderIdentity, QueryIdentity: request.QueryIdentity, Identity: request.Identity, DisplayName: request.DisplayName, Description: request.Description, ViewType: request.ViewType, Source: request.Source, InputSchema: request.InputSchema, OutputSchema: request.OutputSchema, Meta: request.Meta, Active: request.Active})
	if err != nil {
		return transport.RespondDomainError(c, h.logger, err)
	}
	return c.Status(fiber.StatusCreated).JSON(h.response(value, project))
}

// List godoc
// @Summary Список DataView
// @Description Возвращает активные DataView проекта с optional фильтрами папки и Query. DataView, связанные с soft-deleted Query, не возвращаются.
// @Tags data-views
// @Produce json
// @Param project_identity path string true "Project identity" example(demo-project)
// @Param folder_identity query string false "Folder identity" example(root-data-views)
// @Param query_identity query string false "Query identity" example(users-list)
// @Success 200 {object} DataViewsListResponse
// @Failure 400 {object} transport.ErrorResponse
// @Failure 404 {object} transport.ErrorResponse
// @Failure 500 {object} transport.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{project_identity}/data-views [get]
func (h *Handler) List(c *fiber.Ctx) error {
	project := c.Params("project_identity")
	var folder, queryIdentity *string
	if value := c.Query("folder_identity"); value != "" {
		folder = &value
	}
	if value := c.Query("query_identity"); value != "" {
		queryIdentity = &value
	}
	values, err := h.service.List(c.UserContext(), adapters.ListDataViewsInput{ProjectIdentity: project, FolderIdentity: folder, QueryIdentity: queryIdentity})
	if err != nil {
		return transport.RespondDomainError(c, h.logger, err)
	}
	items := make([]*DataViewResponse, 0, len(values))
	for _, value := range values {
		items = append(items, h.response(value, project))
	}
	return c.JSON(DataViewsListResponse{Items: items})
}

// GetByIdentity godoc
// @Summary Получить DataView
// @Description Возвращает активный DataView по identity в пределах проекта вместе с identity папки и связанной Query.
// @Tags data-views
// @Produce json
// @Param project_identity path string true "Project identity" example(demo-project)
// @Param data_view_identity path string true "DataView identity" example(users-table-view)
// @Success 200 {object} DataViewResponse
// @Failure 400 {object} transport.ErrorResponse
// @Failure 404 {object} transport.ErrorResponse
// @Failure 500 {object} transport.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{project_identity}/data-views/{data_view_identity} [get]
func (h *Handler) GetByIdentity(c *fiber.Ctx) error {
	project := c.Params("project_identity")
	value, err := h.service.GetByIdentity(c.UserContext(), adapters.GetDataViewInput{ProjectIdentity: project, DataViewIdentity: c.Params("data_view_identity")})
	if err != nil {
		return transport.RespondDomainError(c, h.logger, err)
	}
	return c.JSON(h.response(value, project))
}

// Update godoc
// @Summary Обновить DataView
// @Description Полностью заменяет editable payload DataView, включая папку, связанную Query и authoring configuration. Поля id, identity, createdAt и deletedAt сохраняются.
// @Tags data-views
// @Accept json
// @Produce json
// @Param project_identity path string true "Project identity" example(demo-project)
// @Param data_view_identity path string true "DataView identity" example(users-table-view)
// @Param request body UpdateDataViewRequest true "Параметры обновления DataView"
// @Success 200 {object} DataViewResponse
// @Failure 400 {object} transport.ErrorResponse
// @Failure 404 {object} transport.ErrorResponse
// @Failure 500 {object} transport.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{project_identity}/data-views/{data_view_identity} [patch]
func (h *Handler) Update(c *fiber.Ctx) error {
	var request UpdateDataViewRequest
	if err := c.BodyParser(&request); err != nil {
		return transport.WriteErrorResponse(c, transport.ErrInvalidBody)
	}
	if err := h.validator.Validate(&request); err != nil {
		return transport.WriteErrorResponse(c, transport.ErrValidationError)
	}
	project := c.Params("project_identity")
	value, err := h.service.Update(c.UserContext(), adapters.UpdateDataViewInput{ProjectIdentity: project, DataViewIdentity: c.Params("data_view_identity"), FolderIdentity: request.FolderIdentity, QueryIdentity: request.QueryIdentity, DisplayName: request.DisplayName, Description: request.Description, ViewType: request.ViewType, Source: request.Source, InputSchema: request.InputSchema, OutputSchema: request.OutputSchema, Meta: request.Meta, Active: request.Active})
	if err != nil {
		return transport.RespondDomainError(c, h.logger, err)
	}
	return c.JSON(h.response(value, project))
}

// SoftDelete godoc
// @Summary Удалить DataView
// @Description Выполняет soft-delete DataView по identity в пределах проекта.
// @Tags data-views
// @Param project_identity path string true "Project identity"
// @Param data_view_identity path string true "DataView identity"
// @Success 204
// @Failure 400 {object} transport.ErrorResponse
// @Failure 404 {object} transport.ErrorResponse
// @Failure 500 {object} transport.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{project_identity}/data-views/{data_view_identity} [delete]
func (h *Handler) SoftDelete(c *fiber.Ctx) error { return h.change(c, h.service.SoftDelete) }

// Restore godoc
// @Summary Восстановить DataView
// @Description Восстанавливает soft-deleted DataView по identity в пределах проекта.
// @Tags data-views
// @Param project_identity path string true "Project identity"
// @Param data_view_identity path string true "DataView identity"
// @Success 204
// @Failure 400 {object} transport.ErrorResponse
// @Failure 404 {object} transport.ErrorResponse
// @Failure 500 {object} transport.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{project_identity}/data-views/{data_view_identity}/restore [post]
func (h *Handler) Restore(c *fiber.Ctx) error { return h.change(c, h.service.Restore) }

// HardDelete godoc
// @Summary Физически удалить DataView
// @Description Выполняет hard-delete soft-deleted DataView по identity в пределах проекта.
// @Tags data-views
// @Param project_identity path string true "Project identity"
// @Param data_view_identity path string true "DataView identity"
// @Success 204
// @Failure 400 {object} transport.ErrorResponse
// @Failure 404 {object} transport.ErrorResponse
// @Failure 500 {object} transport.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{project_identity}/data-views/{data_view_identity}/hard [delete]
func (h *Handler) HardDelete(c *fiber.Ctx) error { return h.change(c, h.service.HardDelete) }
