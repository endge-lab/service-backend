package data_view

import (
	httpobservability "github.com/endge-lab/service-backend/internal/api/http/observability"
	respond "github.com/endge-lab/service-backend/internal/api/http/respond"
	"github.com/endge-lab/service-backend/internal/observability"
	"github.com/endge-lab/service-backend/internal/usecase/data_views"
	appvalidator "github.com/endge-lab/service-kit-go/pkg/validator"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service   UseCase
	validator appvalidator.Validator
	observer  observability.Observer
}

func NewHandler(s UseCase, v appvalidator.Validator, core *observability.Core, metrics *httpobservability.HandlerMetrics) *Handler {
	observer := core.For(observability.LayerHandler, "data_view_http_handler").WithRecorder(metrics)
	return &Handler{service: s, validator: v, observer: observer}
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
// @Failure 400 {object} respond.ErrorResponse
// @Failure 404 {object} respond.ErrorResponse
// @Failure 409 {object} respond.ErrorResponse
// @Failure 500 {object} respond.ErrorResponse
// @Param X-Endge-Workspace header string true "Workspace identity" example(demo-workspace)
// @Security BearerAuth && WorkspaceAuth
// @Router /api/v1/projects/{project_identity}/data-views [post]
func (h *Handler) Create(c *fiber.Ctx) error {
	var request CreateDataViewRequest
	if err := c.BodyParser(&request); err != nil {
		return respond.WriteErrorResponse(c, respond.ErrInvalidBody)
	}
	if err := h.validator.Validate(&request); err != nil {
		return respond.WriteErrorResponse(c, respond.ErrValidationError)
	}
	project := c.Params("project_identity")
	value, err := h.service.Create(c.UserContext(), data_views.CreateDataViewInput{ProjectIdentity: project, FolderIdentity: request.FolderIdentity, QueryIdentity: request.QueryIdentity, Identity: request.Identity, DisplayName: request.DisplayName, Description: request.Description, ViewType: request.ViewType, Source: request.Source, InputSchema: request.InputSchema, OutputSchema: request.OutputSchema, Meta: request.Meta, Active: request.Active})
	if err != nil {
		return respond.RespondDomainError(c, h.observer.Logger(), err)
	}
	return c.Status(fiber.StatusCreated).JSON(h.response(value, project))
}

// List godoc
// @Summary Список DataView
// @Description Возвращает активные DataView проекта с optional фильтрами папки и Query. DataView, связанные с soft-deleted Query, не возвращаются.
// @Tags data-views
// @Produce json
// @Param project_identity path string true "Project identity" example(demo-project)
// @Param folder_identity query string false "Folder identity" example(shared-data-views)
// @Param query_identity query string false "Query identity" example(users-list)
// @Success 200 {object} DataViewsListResponse
// @Failure 400 {object} respond.ErrorResponse
// @Failure 404 {object} respond.ErrorResponse
// @Failure 500 {object} respond.ErrorResponse
// @Param X-Endge-Workspace header string true "Workspace identity" example(demo-workspace)
// @Security BearerAuth && WorkspaceAuth
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
	values, err := h.service.List(c.UserContext(), data_views.ListDataViewsInput{ProjectIdentity: project, FolderIdentity: folder, QueryIdentity: queryIdentity})
	if err != nil {
		return respond.RespondDomainError(c, h.observer.Logger(), err)
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
// @Param data_view_identity path string true "DataView identity" example(users-table)
// @Success 200 {object} DataViewResponse
// @Failure 400 {object} respond.ErrorResponse
// @Failure 404 {object} respond.ErrorResponse
// @Failure 500 {object} respond.ErrorResponse
// @Param X-Endge-Workspace header string true "Workspace identity" example(demo-workspace)
// @Security BearerAuth && WorkspaceAuth
// @Router /api/v1/projects/{project_identity}/data-views/{data_view_identity} [get]
func (h *Handler) GetByIdentity(c *fiber.Ctx) error {
	project := c.Params("project_identity")
	value, err := h.service.GetByIdentity(c.UserContext(), data_views.GetDataViewInput{ProjectIdentity: project, DataViewIdentity: c.Params("data_view_identity")})
	if err != nil {
		return respond.RespondDomainError(c, h.observer.Logger(), err)
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
// @Param data_view_identity path string true "DataView identity" example(users-table)
// @Param request body UpdateDataViewRequest true "Параметры обновления DataView"
// @Success 200 {object} DataViewResponse
// @Failure 400 {object} respond.ErrorResponse
// @Failure 404 {object} respond.ErrorResponse
// @Failure 500 {object} respond.ErrorResponse
// @Param X-Endge-Workspace header string true "Workspace identity" example(demo-workspace)
// @Security BearerAuth && WorkspaceAuth
// @Router /api/v1/projects/{project_identity}/data-views/{data_view_identity} [patch]
func (h *Handler) Update(c *fiber.Ctx) error {
	var request UpdateDataViewRequest
	if err := c.BodyParser(&request); err != nil {
		return respond.WriteErrorResponse(c, respond.ErrInvalidBody)
	}
	if err := h.validator.Validate(&request); err != nil {
		return respond.WriteErrorResponse(c, respond.ErrValidationError)
	}
	project := c.Params("project_identity")
	value, err := h.service.Update(c.UserContext(), data_views.UpdateDataViewInput{ProjectIdentity: project, DataViewIdentity: c.Params("data_view_identity"), FolderIdentity: request.FolderIdentity, QueryIdentity: request.QueryIdentity, DisplayName: request.DisplayName, Description: request.Description, ViewType: request.ViewType, Source: request.Source, InputSchema: request.InputSchema, OutputSchema: request.OutputSchema, Meta: request.Meta, Active: request.Active})
	if err != nil {
		return respond.RespondDomainError(c, h.observer.Logger(), err)
	}
	return c.JSON(h.response(value, project))
}

// SoftDelete godoc
// @Summary Удалить DataView
// @Description Выполняет soft-delete DataView по identity в пределах проекта.
// @Tags data-views
// @Param project_identity path string true "Project identity" example(demo-project)
// @Param data_view_identity path string true "DataView identity" example(users-table)
// @Success 204
// @Failure 400 {object} respond.ErrorResponse
// @Failure 404 {object} respond.ErrorResponse
// @Failure 500 {object} respond.ErrorResponse
// @Param X-Endge-Workspace header string true "Workspace identity" example(demo-workspace)
// @Security BearerAuth && WorkspaceAuth
// @Router /api/v1/projects/{project_identity}/data-views/{data_view_identity} [delete]
func (h *Handler) SoftDelete(c *fiber.Ctx) error { return h.change(c, h.service.SoftDelete) }

// Restore godoc
// @Summary Восстановить DataView
// @Description Восстанавливает soft-deleted DataView по identity в пределах проекта.
// @Tags data-views
// @Param project_identity path string true "Project identity" example(demo-project)
// @Param data_view_identity path string true "DataView identity" example(restore-users-table)
// @Success 204
// @Failure 400 {object} respond.ErrorResponse
// @Failure 404 {object} respond.ErrorResponse
// @Failure 500 {object} respond.ErrorResponse
// @Param X-Endge-Workspace header string true "Workspace identity" example(demo-workspace)
// @Security BearerAuth && WorkspaceAuth
// @Router /api/v1/projects/{project_identity}/data-views/{data_view_identity}/restore [post]
func (h *Handler) Restore(c *fiber.Ctx) error { return h.change(c, h.service.Restore) }

// HardDelete godoc
// @Summary Физически удалить DataView
// @Description Выполняет hard-delete soft-deleted DataView по identity в пределах проекта.
// @Tags data-views
// @Param project_identity path string true "Project identity" example(demo-project)
// @Param data_view_identity path string true "DataView identity" example(hard-delete-users-table)
// @Success 204
// @Failure 400 {object} respond.ErrorResponse
// @Failure 404 {object} respond.ErrorResponse
// @Failure 500 {object} respond.ErrorResponse
// @Param X-Endge-Workspace header string true "Workspace identity" example(demo-workspace)
// @Security BearerAuth && WorkspaceAuth
// @Router /api/v1/projects/{project_identity}/data-views/{data_view_identity}/hard [delete]
func (h *Handler) HardDelete(c *fiber.Ctx) error { return h.change(c, h.service.HardDelete) }
