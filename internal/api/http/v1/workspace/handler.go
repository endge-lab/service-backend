package http

import (
	httpobservability "github.com/endge-lab/service-backend/internal/api/http/observability"
	respond "github.com/endge-lab/service-backend/internal/api/http/respond"
	"github.com/endge-lab/service-backend/internal/observability"
	"github.com/endge-lab/service-backend/internal/usecase/workspaces"
	appvalidator "github.com/endge-lab/service-kit-go/pkg/validator"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service   UseCase
	validator appvalidator.Validator
	observer  observability.Observer
}

func NewHandler(s UseCase, v appvalidator.Validator, core *observability.Core, metrics *httpobservability.HandlerMetrics) *Handler {
	observer := core.For(observability.LayerHandler, "workspace_http_handler").WithRecorder(metrics)
	return &Handler{service: s, validator: v, observer: observer}
}

// Create godoc
// @Summary Создать workspace
// @Description Создаёт корневой workspace. Configuration необязательна: при отсутствии backend применяет system default. Endpoint не требует X-Endge-Workspace.
// @Tags workspaces
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateRequest true "Данные workspace"
// @Success 201 {object} Response
// @Failure 400 {object} respond.ErrorResponse
// @Failure 409 {object} respond.ErrorResponse
// @Failure 500 {object} respond.ErrorResponse
// @Router /api/v1/workspaces [post]
func (h *Handler) Create(c *fiber.Ctx) error {
	var r CreateRequest
	if err := c.BodyParser(&r); err != nil {
		return respond.WriteErrorResponse(c, respond.ErrInvalidBody)
	}
	if err := h.validator.Validate(&r); err != nil {
		return respond.WriteErrorResponse(c, respond.ErrValidationError)
	}
	v, err := h.service.Create(c.UserContext(), workspaces.CreateWorkspaceInput{Identity: r.Identity, DisplayName: r.DisplayName, Configuration: r.Configuration})
	if err != nil {
		return respond.RespondDomainError(c, h.observer.Logger(), err)
	}
	return c.Status(201).JSON(response(v))
}

// List godoc
// @Summary Список workspaces
// @Description Возвращает все workspaces без user/membership filtering. Endpoint не требует X-Endge-Workspace; sse.manualToken redacted.
// @Tags workspaces
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string][]Response
// @Failure 500 {object} respond.ErrorResponse
// @Router /api/v1/workspaces [get]
func (h *Handler) List(c *fiber.Ctx) error {
	v, err := h.service.List(c.UserContext())
	if err != nil {
		return respond.RespondDomainError(c, h.observer.Logger(), err)
	}
	items := make([]*Response, 0, len(v))
	for _, x := range v {
		items = append(items, response(x))
	}
	return c.JSON(fiber.Map{"items": items})
}

// Get godoc
// @Summary Получить workspace
// @Description Возвращает workspace по identity с полной root configuration. sse.manualToken не возвращается.
// @Tags workspaces
// @Produce json
// @Security BearerAuth
// @Param workspace_identity path string true "Workspace identity" example(demo-workspace)
// @Success 200 {object} Response
// @Failure 400 {object} respond.ErrorResponse
// @Failure 404 {object} respond.ErrorResponse
// @Failure 500 {object} respond.ErrorResponse
// @Router /api/v1/workspaces/{workspace_identity} [get]
func (h *Handler) Get(c *fiber.Ctx) error {
	v, err := h.service.GetByIdentity(c.UserContext(), c.Params("workspace_identity"))
	if err != nil {
		return respond.RespondDomainError(c, h.observer.Logger(), err)
	}
	return c.JSON(response(v))
}

// Update godoc
// @Summary Обновить workspace
// @Description Частично обновляет верхнеуровневые поля. Переданная configuration полностью заменяет root configuration; JSON merge не выполняется. sse.manualToken redacted в response.
// @Tags workspaces
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param workspace_identity path string true "Workspace identity" example(demo-workspace)
// @Param request body UpdateRequest true "Изменяемые поля workspace"
// @Success 200 {object} Response
// @Failure 400 {object} respond.ErrorResponse
// @Failure 404 {object} respond.ErrorResponse
// @Failure 500 {object} respond.ErrorResponse
// @Router /api/v1/workspaces/{workspace_identity} [patch]
func (h *Handler) Update(c *fiber.Ctx) error {
	var r UpdateRequest
	if err := c.BodyParser(&r); err != nil {
		return respond.WriteErrorResponse(c, respond.ErrInvalidBody)
	}
	v, err := h.service.Update(c.UserContext(), workspaces.UpdateWorkspaceInput{Identity: c.Params("workspace_identity"), DisplayName: r.DisplayName, Configuration: r.Configuration})
	if err != nil {
		return respond.RespondDomainError(c, h.observer.Logger(), err)
	}
	return c.JSON(response(v))
}
