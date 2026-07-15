package query

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
	service   adapters.QueryService
	validator appvalidator.Validator
	logger    *zap.Logger
	tracer    trace.Tracer
}

func NewHandler(s *usecase.Service, v appvalidator.Validator, l *zap.Logger, t trace.Tracer) *Handler {
	return &Handler{service: s.Queries, validator: v, logger: l.With(zap.String("component", "query_http_handler")), tracer: t}
}

// Create godoc
// @Summary Создать Query
// @Description Создает конфигурацию Query в папке проекта. Query source, headers, параметры и mock data только хранятся и не исполняются сервисом.
// @Tags queries
// @Accept json
// @Produce json
// @Param project_identity path string true "Project identity" example(demo-project)
// @Param request body CreateQueryRequest true "Параметры Query"
// @Success 201 {object} QueryResponse
// @Failure 400 {object} transport.ErrorResponse
// @Failure 404 {object} transport.ErrorResponse
// @Failure 409 {object} transport.ErrorResponse
// @Failure 500 {object} transport.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{project_identity}/queries [post]
func (h *Handler) Create(c *fiber.Ctx) error {
	var request CreateQueryRequest
	if err := c.BodyParser(&request); err != nil {
		return transport.WriteErrorResponse(c, transport.ErrInvalidBody)
	}
	if err := h.validator.Validate(&request); err != nil {
		return transport.WriteErrorResponse(c, transport.ErrValidationError)
	}
	project := c.Params("project_identity")
	value, err := h.service.Create(c.UserContext(), adapters.CreateQueryInput{ProjectIdentity: project, FolderIdentity: request.FolderIdentity, Identity: request.Identity, DisplayName: request.DisplayName, Description: request.Description, QueryType: request.QueryType, Source: request.Source, Params: request.Params, Headers: request.Headers, Auth: request.Auth, TimeoutMS: request.TimeoutMS, MockData: request.MockData, MockDataEnabled: request.MockDataEnabled, Meta: request.Meta, Active: request.Active})
	if err != nil {
		return transport.RespondDomainError(c, h.logger, err)
	}
	return c.Status(fiber.StatusCreated).JSON(h.response(value, project))
}

// List godoc
// @Summary Список Query
// @Description Возвращает активные Query проекта. Можно отфильтровать записи по identity папки и query type; soft-deleted Query не возвращаются.
// @Tags queries
// @Produce json
// @Param project_identity path string true "Project identity" example(demo-project)
// @Param folder_identity query string false "Folder identity" example(root-queries)
// @Param query_type query string false "Query type" example(http)
// @Success 200 {object} QueriesListResponse
// @Failure 400 {object} transport.ErrorResponse
// @Failure 404 {object} transport.ErrorResponse
// @Failure 500 {object} transport.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{project_identity}/queries [get]
func (h *Handler) List(c *fiber.Ctx) error {
	project := c.Params("project_identity")
	var folder, queryType *string
	if value := c.Query("folder_identity"); value != "" {
		folder = &value
	}
	if value := c.Query("query_type"); value != "" {
		queryType = &value
	}
	values, err := h.service.List(c.UserContext(), adapters.ListQueriesInput{ProjectIdentity: project, FolderIdentity: folder, QueryType: queryType})
	if err != nil {
		return transport.RespondDomainError(c, h.logger, err)
	}
	items := make([]*QueryResponse, 0, len(values))
	for _, value := range values {
		items = append(items, h.response(value, project))
	}
	return c.JSON(QueriesListResponse{Items: items})
}

// GetByIdentity godoc
// @Summary Получить Query
// @Description Возвращает активную Query по identity в пределах проекта вместе с identity ее папки.
// @Tags queries
// @Produce json
// @Param project_identity path string true "Project identity" example(demo-project)
// @Param query_identity path string true "Query identity" example(users-list)
// @Success 200 {object} QueryResponse
// @Failure 400 {object} transport.ErrorResponse
// @Failure 404 {object} transport.ErrorResponse
// @Failure 500 {object} transport.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{project_identity}/queries/{query_identity} [get]
func (h *Handler) GetByIdentity(c *fiber.Ctx) error {
	project := c.Params("project_identity")
	value, err := h.service.GetByIdentity(c.UserContext(), adapters.GetQueryInput{ProjectIdentity: project, QueryIdentity: c.Params("query_identity")})
	if err != nil {
		return transport.RespondDomainError(c, h.logger, err)
	}
	return c.JSON(h.response(value, project))
}

// Update godoc
// @Summary Обновить Query
// @Description Полностью заменяет editable payload Query, включая папку, source, параметры, headers, auth и mock data. Поля id, identity, createdAt и deletedAt сохраняются.
// @Tags queries
// @Accept json
// @Produce json
// @Param project_identity path string true "Project identity" example(demo-project)
// @Param query_identity path string true "Query identity" example(users-list)
// @Param request body UpdateQueryRequest true "Параметры обновления Query"
// @Success 200 {object} QueryResponse
// @Failure 400 {object} transport.ErrorResponse
// @Failure 404 {object} transport.ErrorResponse
// @Failure 500 {object} transport.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{project_identity}/queries/{query_identity} [patch]
func (h *Handler) Update(c *fiber.Ctx) error {
	var request UpdateQueryRequest
	if err := c.BodyParser(&request); err != nil {
		return transport.WriteErrorResponse(c, transport.ErrInvalidBody)
	}
	if err := h.validator.Validate(&request); err != nil {
		return transport.WriteErrorResponse(c, transport.ErrValidationError)
	}
	project := c.Params("project_identity")
	value, err := h.service.Update(c.UserContext(), adapters.UpdateQueryInput{ProjectIdentity: project, QueryIdentity: c.Params("query_identity"), FolderIdentity: request.FolderIdentity, DisplayName: request.DisplayName, Description: request.Description, QueryType: request.QueryType, Source: request.Source, Params: request.Params, Headers: request.Headers, Auth: request.Auth, TimeoutMS: request.TimeoutMS, MockData: request.MockData, MockDataEnabled: request.MockDataEnabled, Meta: request.Meta, Active: request.Active})
	if err != nil {
		return transport.RespondDomainError(c, h.logger, err)
	}
	return c.JSON(h.response(value, project))
}

// SoftDelete godoc
// @Summary Удалить Query
// @Description Выполняет soft-delete Query. Связанные DataView физически не удаляются, но не попадают в обычные query-based сценарии.
// @Tags queries
// @Param project_identity path string true "Project identity"
// @Param query_identity path string true "Query identity"
// @Success 204
// @Failure 400 {object} transport.ErrorResponse
// @Failure 404 {object} transport.ErrorResponse
// @Failure 500 {object} transport.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{project_identity}/queries/{query_identity} [delete]
func (h *Handler) SoftDelete(c *fiber.Ctx) error { return h.change(c, h.service.SoftDelete) }

// Restore godoc
// @Summary Восстановить Query
// @Description Восстанавливает soft-deleted Query по identity в пределах проекта.
// @Tags queries
// @Param project_identity path string true "Project identity"
// @Param query_identity path string true "Query identity"
// @Success 204
// @Failure 400 {object} transport.ErrorResponse
// @Failure 404 {object} transport.ErrorResponse
// @Failure 500 {object} transport.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{project_identity}/queries/{query_identity}/restore [post]
func (h *Handler) Restore(c *fiber.Ctx) error { return h.change(c, h.service.Restore) }

// HardDelete godoc
// @Summary Физически удалить Query
// @Description Выполняет hard-delete soft-deleted Query. Связанные DataView удаляются каскадно на уровне базы данных.
// @Tags queries
// @Param project_identity path string true "Project identity"
// @Param query_identity path string true "Query identity"
// @Success 204
// @Failure 400 {object} transport.ErrorResponse
// @Failure 404 {object} transport.ErrorResponse
// @Failure 500 {object} transport.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{project_identity}/queries/{query_identity}/hard [delete]
func (h *Handler) HardDelete(c *fiber.Ctx) error { return h.change(c, h.service.HardDelete) }
