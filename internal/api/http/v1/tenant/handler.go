package tenant

import (
	httpobservability "github.com/endge-lab/service-backend/internal/api/http/observability"
	respond "github.com/endge-lab/service-backend/internal/api/http/respond"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/observability"
	"github.com/endge-lab/service-backend/internal/usecase/tenants"
	appvalidator "github.com/endge-lab/service-kit-go/pkg/validator"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service   UseCase
	validator appvalidator.Validator
	observer  observability.Observer
}

func NewHandler(service UseCase, validator appvalidator.Validator, core *observability.Core, metrics *httpobservability.HandlerMetrics) *Handler {
	return &Handler{service: service, validator: validator, observer: core.For(observability.LayerHandler, "tenant_http_handler").WithRecorder(metrics)}
}

// CreateTenant godoc
// @Summary Создать tenant
// @Description Создаёт final configuration layer Tenant в workspace из X-Endge-Workspace. При отсутствии folderIdentity используется root-tenants.
// @Tags tenants
// @Accept json
// @Produce json
// @Param X-Endge-Workspace header string true "Workspace identity" example(demo-workspace)
// @Param request body CreateTenantRequest true "Данные tenant"
// @Success 201 {object} TenantResponse
// @Failure 400 {object} respond.ErrorResponse
// @Failure 404 {object} respond.ErrorResponse
// @Failure 409 {object} respond.ErrorResponse
// @Failure 500 {object} respond.ErrorResponse
// @Security BearerAuth && WorkspaceAuth
// @Router /api/v1/tenants [post]
func (h *Handler) CreateTenant(c *fiber.Ctx) error {
	var request CreateTenantRequest
	if err := c.BodyParser(&request); err != nil {
		return respond.WriteErrorResponse(c, respond.ErrInvalidBody)
	}
	if err := h.validator.Validate(&request); err != nil {
		return respond.WriteErrorResponse(c, respond.ErrValidationError)
	}
	var configuration *entities.EndgeConfigurationContribution
	if request.Configuration != nil {
		var err error
		configuration, err = request.Configuration.domain()
		if err != nil {
			return respond.WriteErrorResponse(c, respond.ErrInvalidBody)
		}
	}
	value, err := h.service.Create(c.UserContext(), tenants.CreateTenantInput{Identity: request.Identity, DisplayName: request.DisplayName, Code: request.Code, Description: request.Description, FolderIdentity: request.FolderIdentity, Configuration: configuration})
	if err != nil {
		return respond.RespondDomainError(c, h.observer.Logger(), err)
	}
	return c.Status(fiber.StatusCreated).JSON(response(value))
}

// ListTenants godoc
// @Summary Список tenants
// @Description Возвращает tenants текущего workspace; folder_identity ограничивает список папкой типа tenants.
// @Tags tenants
// @Produce json
// @Param X-Endge-Workspace header string true "Workspace identity" example(demo-workspace)
// @Param folder_identity query string false "Tenant folder identity" example(root-tenants)
// @Success 200 {object} TenantsListResponse
// @Failure 400 {object} respond.ErrorResponse
// @Failure 404 {object} respond.ErrorResponse
// @Failure 500 {object} respond.ErrorResponse
// @Security BearerAuth && WorkspaceAuth
// @Router /api/v1/tenants [get]
func (h *Handler) ListTenants(c *fiber.Ctx) error {
	var folderIdentity *string
	if value, ok := c.Queries()["folder_identity"]; ok {
		folderIdentity = &value
	}
	items, err := h.service.List(c.UserContext(), tenants.ListTenantsInput{FolderIdentity: folderIdentity})
	if err != nil {
		return respond.RespondDomainError(c, h.observer.Logger(), err)
	}
	return c.JSON(listResponse(items))
}

// GetTenantByIdentity godoc
// @Summary Получить tenant
// @Description Возвращает tenant по identity только из workspace X-Endge-Workspace.
// @Tags tenants
// @Produce json
// @Param X-Endge-Workspace header string true "Workspace identity" example(demo-workspace)
// @Param tenant_identity path string true "Tenant identity" example(tenant-default)
// @Success 200 {object} TenantResponse
// @Failure 400 {object} respond.ErrorResponse
// @Failure 404 {object} respond.ErrorResponse
// @Failure 500 {object} respond.ErrorResponse
// @Security BearerAuth && WorkspaceAuth
// @Router /api/v1/tenants/{tenant_identity} [get]
func (h *Handler) GetTenantByIdentity(c *fiber.Ctx) error {
	value, err := h.service.GetByIdentity(c.UserContext(), c.Params("tenant_identity"))
	if err != nil {
		return respond.RespondDomainError(c, h.observer.Logger(), err)
	}
	return c.JSON(response(value))
}

// UpdateTenant godoc
// @Summary Обновить tenant
// @Description Частично обновляет tenant. folderIdentity:null перемещает tenant в root-tenants; переданная configuration полностью заменяет contribution.
// @Tags tenants
// @Accept json
// @Produce json
// @Param X-Endge-Workspace header string true "Workspace identity" example(demo-workspace)
// @Param tenant_identity path string true "Tenant identity" example(tenant-default)
// @Param request body UpdateTenantRequest true "Поля PATCH tenant"
// @Success 200 {object} TenantResponse
// @Failure 400 {object} respond.ErrorResponse
// @Failure 404 {object} respond.ErrorResponse
// @Failure 409 {object} respond.ErrorResponse
// @Failure 500 {object} respond.ErrorResponse
// @Security BearerAuth && WorkspaceAuth
// @Router /api/v1/tenants/{tenant_identity} [patch]
func (h *Handler) UpdateTenant(c *fiber.Ctx) error {
	var request UpdateTenantRequest
	if err := c.BodyParser(&request); err != nil {
		return respond.WriteErrorResponse(c, respond.ErrInvalidBody)
	}
	if err := h.validator.Validate(&request); err != nil {
		return respond.WriteErrorResponse(c, respond.ErrValidationError)
	}
	input, err := request.input(c.Params("tenant_identity"))
	if err != nil {
		return respond.WriteErrorResponse(c, respond.ErrInvalidBody)
	}
	value, err := h.service.Update(c.UserContext(), input)
	if err != nil {
		return respond.RespondDomainError(c, h.observer.Logger(), err)
	}
	return c.JSON(response(value))
}

// HardDeleteTenant godoc
// @Summary Удалить tenant
// @Description Физически удаляет tenant по identity в workspace X-Endge-Workspace.
// @Tags tenants
// @Param X-Endge-Workspace header string true "Workspace identity" example(demo-workspace)
// @Param tenant_identity path string true "Tenant identity" example(tenant-default)
// @Success 204
// @Failure 400 {object} respond.ErrorResponse
// @Failure 404 {object} respond.ErrorResponse
// @Failure 500 {object} respond.ErrorResponse
// @Security BearerAuth && WorkspaceAuth
// @Router /api/v1/tenants/{tenant_identity} [delete]
func (h *Handler) HardDeleteTenant(c *fiber.Ctx) error {
	if err := h.service.HardDelete(c.UserContext(), c.Params("tenant_identity")); err != nil {
		return respond.RespondDomainError(c, h.observer.Logger(), err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
