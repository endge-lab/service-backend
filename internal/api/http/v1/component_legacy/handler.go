package component_legacy

import (
	respond "github.com/endge-lab/service-backend/internal/api/http/respond"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/components_legacy"
	appvalidator "github.com/endge-lab/service-kit-go/pkg/validator"
	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type Handler struct {
	service   UseCase
	validator appvalidator.Validator
	logger    *zap.Logger
	tracer    trace.Tracer
}

func NewHandler(s UseCase, v appvalidator.Validator, l *zap.Logger, t trace.Tracer) *Handler {
	return &Handler{service: s, validator: v, logger: l.With(zap.String("component", "component_legacy_http_handler")), tracer: t}
}

// Create godoc
// @Summary Создать компонент
// @Description Создает component-sfc в папке проекта. Source хранится как authoring source и не компилируется и не выполняется.
// @Tags components-legacy
// @Accept json
// @Produce json
// @Param project_identity path string true "Project identity" example(demo-project)
// @Param request body CreateComponentLegacyRequest true "Параметры компонента"
// @Success 201 {object} ComponentLegacyResponse
// @Failure 400 {object} respond.ErrorResponse
// @Failure 404 {object} respond.ErrorResponse
// @Failure 409 {object} respond.ErrorResponse
// @Failure 500 {object} respond.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{project_identity}/components-legacy [post]
func (h *Handler) Create(c *fiber.Ctx) error {
	var r CreateComponentLegacyRequest
	if e := c.BodyParser(&r); e != nil {
		return respond.WriteErrorResponse(c, respond.ErrInvalidBody)
	}
	if e := h.validator.Validate(&r); e != nil {
		return respond.WriteErrorResponse(c, respond.ErrValidationError)
	}
	p := c.Params("project_identity")
	v, e := h.service.Create(c.UserContext(), components_legacy.CreateComponentLegacyInput{ProjectIdentity: p, FolderIdentity: r.FolderIdentity, Identity: r.Identity, DisplayName: r.DisplayName, Description: r.Description, ComponentType: r.ComponentType, Source: r.Source, PropsSchema: r.PropsSchema, Bindings: r.Bindings, Meta: r.Meta, Active: r.Active})
	if e != nil {
		return respond.RespondDomainError(c, h.logger, e)
	}
	return c.Status(201).JSON(h.response(v, p))
}

// GetByIdentity godoc
// @Summary Получить компонент
// @Description Возвращает активный компонент по identity в пределах проекта.
// @Tags components-legacy
// @Produce json
// @Param project_identity path string true "Project identity" example(demo-project)
// @Param component_identity path string true "Legacy component identity" example(user-card)
// @Success 200 {object} ComponentLegacyResponse
// @Failure 400 {object} respond.ErrorResponse
// @Failure 404 {object} respond.ErrorResponse
// @Failure 500 {object} respond.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{project_identity}/components-legacy/{component_identity} [get]
func (h *Handler) GetByIdentity(c *fiber.Ctx) error {
	p := c.Params("project_identity")
	v, e := h.service.GetByIdentity(c.UserContext(), components_legacy.GetComponentLegacyInput{ProjectIdentity: p, ComponentLegacyIdentity: c.Params("component_identity")})
	if e != nil {
		return respond.RespondDomainError(c, h.logger, e)
	}
	return c.JSON(h.response(v, p))
}

// List godoc
// @Summary Список компонентов
// @Description Возвращает неудаленные компоненты проекта с optional фильтрами папки и типа.
// @Tags components-legacy
// @Produce json
// @Param project_identity path string true "Project identity" example(demo-project)
// @Param folder_identity query string false "Folder identity" example(root-components-legacy)
// @Param component_type query string false "Legacy component type" Enums(component-sfc)
// @Success 200 {object} ComponentsLegacyListResponse
// @Failure 400 {object} respond.ErrorResponse
// @Failure 404 {object} respond.ErrorResponse
// @Failure 500 {object} respond.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{project_identity}/components-legacy [get]
func (h *Handler) List(c *fiber.Ctx) error {
	p := c.Params("project_identity")
	var folder *string
	if q := c.Query("folder_identity"); q != "" {
		folder = &q
	}
	var typ *entities.RComponentLegacyType
	if q := c.Query("component_type"); q != "" {
		v := entities.RComponentLegacyType(q)
		typ = &v
	}
	values, e := h.service.List(c.UserContext(), components_legacy.ListComponentsLegacyInput{ProjectIdentity: p, FolderIdentity: folder, ComponentType: typ})
	if e != nil {
		return respond.RespondDomainError(c, h.logger, e)
	}
	items := make([]*ComponentLegacyResponse, 0, len(values))
	for _, value := range values {
		items = append(items, h.response(value, p))
	}
	return c.JSON(ComponentsLegacyListResponse{Items: items})
}

// Update godoc
// @Summary Обновить компонент
// @Description Заменяет editable payload компонента, сохраняя id, identity, createdAt и deletedAt.
// @Tags components-legacy
// @Accept json
// @Produce json
// @Param project_identity path string true "Project identity" example(demo-project)
// @Param component_identity path string true "Legacy component identity" example(user-card)
// @Param request body UpdateComponentLegacyRequest true "Параметры обновления компонента"
// @Success 200 {object} ComponentLegacyResponse
// @Failure 400 {object} respond.ErrorResponse
// @Failure 404 {object} respond.ErrorResponse
// @Failure 500 {object} respond.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{project_identity}/components-legacy/{component_identity} [patch]
func (h *Handler) Update(c *fiber.Ctx) error {
	var r UpdateComponentLegacyRequest
	if e := c.BodyParser(&r); e != nil {
		return respond.WriteErrorResponse(c, respond.ErrInvalidBody)
	}
	if e := h.validator.Validate(&r); e != nil {
		return respond.WriteErrorResponse(c, respond.ErrValidationError)
	}
	p := c.Params("project_identity")
	v, e := h.service.Update(c.UserContext(), components_legacy.UpdateComponentLegacyInput{ProjectIdentity: p, ComponentLegacyIdentity: c.Params("component_identity"), FolderIdentity: r.FolderIdentity, DisplayName: r.DisplayName, Description: r.Description, ComponentType: r.ComponentType, Source: r.Source, PropsSchema: r.PropsSchema, Bindings: r.Bindings, Meta: r.Meta, Active: r.Active})
	if e != nil {
		return respond.RespondDomainError(c, h.logger, e)
	}
	return c.JSON(h.response(v, p))
}

// SoftDelete godoc
// @Summary Удалить компонент
// @Description Выполняет soft-delete компонента по identity.
// @Tags components-legacy
// @Param project_identity path string true "Project identity"
// @Param component_identity path string true "Legacy component identity"
// @Success 204
// @Failure 400 {object} respond.ErrorResponse
// @Failure 404 {object} respond.ErrorResponse
// @Failure 500 {object} respond.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{project_identity}/components-legacy/{component_identity} [delete]
func (h *Handler) SoftDelete(c *fiber.Ctx) error { return h.change(c, h.service.SoftDelete) }

// Restore godoc
// @Summary Восстановить компонент
// @Description Восстанавливает soft-deleted компонент по identity.
// @Tags components-legacy
// @Param project_identity path string true "Project identity"
// @Param component_identity path string true "Legacy component identity"
// @Success 204
// @Failure 400 {object} respond.ErrorResponse
// @Failure 404 {object} respond.ErrorResponse
// @Failure 500 {object} respond.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{project_identity}/components-legacy/{component_identity}/restore [post]
func (h *Handler) Restore(c *fiber.Ctx) error { return h.change(c, h.service.Restore) }

// HardDelete godoc
// @Summary Физически удалить компонент
// @Description Выполняет hard-delete компонента по identity.
// @Tags components-legacy
// @Param project_identity path string true "Project identity"
// @Param component_identity path string true "Legacy component identity"
// @Success 204
// @Failure 400 {object} respond.ErrorResponse
// @Failure 404 {object} respond.ErrorResponse
// @Failure 500 {object} respond.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{project_identity}/components-legacy/{component_identity}/hard [delete]
func (h *Handler) HardDelete(c *fiber.Ctx) error { return h.change(c, h.service.HardDelete) }
