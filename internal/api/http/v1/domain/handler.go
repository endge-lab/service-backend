package domain

import (
	"strconv"

	httpobservability "github.com/endge-lab/service-backend/internal/api/http/observability"
	respond "github.com/endge-lab/service-backend/internal/api/http/respond"
	"github.com/endge-lab/service-backend/internal/observability"
	"github.com/endge-lab/service-backend/internal/usecase/dependencies"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service  UseCase
	observer observability.Observer
}

func NewHandler(service UseCase, core *observability.Core, metrics *httpobservability.HandlerMetrics) *Handler {
	return &Handler{service: service, observer: core.For(observability.LayerHandler, "domain_http_handler").WithRecorder(metrics)}
}

// ListUsages godoc
// @Summary Получить usages domain identity
// @Description Возвращает документы текущего workspace, которые ссылаются на dependency identity внутри source или authoring JSON. Dependency index является derived projection и через API не редактируется.
// @Tags domain
// @Produce json
// @Param X-Endge-Workspace header string true "Workspace identity" example(demo-workspace)
// @Param dependency_type query string true "Dependency type" example(type)
// @Param dependency_identity query string true "Dependency identity" example(Orders)
// @Param limit query int false "Размер страницы: от 1 до 200, по умолчанию 50" default(50)
// @Param offset query int false "Смещение, по умолчанию 0" default(0)
// @Success 200 {object} UsagesListResponse
// @Failure 400 {object} respond.ErrorResponse
// @Failure 500 {object} respond.ErrorResponse
// @Security BearerAuth && WorkspaceAuth
// @Router /api/v1/domain/usages [get]
func (h *Handler) ListUsages(c *fiber.Ctx) error {
	input := dependencies.ListUsagesInput{DependencyType: c.Query("dependency_type"), DependencyIdentity: c.Query("dependency_identity")}
	if value, exists := c.Queries()["limit"]; exists {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return respond.WriteErrorResponse(c, respond.ErrValidationError)
		}
		input.Limit = &parsed
	}
	if value, exists := c.Queries()["offset"]; exists {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return respond.WriteErrorResponse(c, respond.ErrValidationError)
		}
		input.Offset = &parsed
	}
	value, err := h.service.ListUsages(c.UserContext(), input)
	if err != nil {
		return respond.RespondDomainError(c, h.observer.Logger(), err)
	}
	return c.JSON(usagesListResponse(value))
}
