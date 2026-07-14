package component

import (
	"context"

	transport "github.com/endge-lab/service-backend/internal/api/http/v1/transport"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase"
	"github.com/endge-lab/service-backend/internal/usecase/adapters"
	servicefiber "github.com/endge-lab/service-kit-go/pkg/httpkit/fiber"
	appvalidator "github.com/endge-lab/service-kit-go/pkg/validator"
	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type Handler struct {
	service   adapters.ComponentService
	folders   adapters.FolderService
	validator appvalidator.Validator
	logger    *zap.Logger
	tracer    trace.Tracer
}

func NewHandler(s *usecase.Service, v appvalidator.Validator, l *zap.Logger, t trace.Tracer) *Handler {
	return &Handler{service: s.Components, folders: s.Folders, validator: v, logger: l, tracer: t}
}
func (h *Handler) response(c *fiber.Ctx, v *entities.Component, p string) (*ComponentResponse, error) {
	f, e := h.folders.GetByID(c.UserContext(), v.FolderID)
	if e != nil {
		return nil, e
	}
	return newComponentResponse(v, p, f.Identity), nil
}

// Create godoc
// @Summary Создать компонент
// @Description Создает component-sfc в папке проекта. Source хранится как authoring source и не компилируется и не выполняется.
// @Tags components
// @Accept json
// @Produce json
// @Param project_identity path string true "Project identity" example(demo-project)
// @Param request body CreateComponentRequest true "Параметры компонента"
// @Success 201 {object} ComponentResponse
// @Failure 400 {object} transport.ErrorResponse
// @Failure 404 {object} transport.ErrorResponse
// @Failure 409 {object} transport.ErrorResponse
// @Failure 500 {object} transport.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{project_identity}/components [post]
func (h *Handler) Create(c *fiber.Ctx) error {
	var r CreateComponentRequest
	if e := c.BodyParser(&r); e != nil {
		return transport.WriteErrorResponse(c, transport.ErrInvalidBody)
	}
	if e := h.validator.Validate(&r); e != nil {
		return transport.WriteErrorResponse(c, transport.ErrValidationError)
	}
	p := c.Params("project_identity")
	v, e := h.service.Create(c.UserContext(), adapters.CreateComponentInput{ProjectIdentity: p, FolderIdentity: r.FolderIdentity, Identity: r.Identity, DisplayName: r.DisplayName, Description: r.Description, ComponentType: r.ComponentType, Source: r.Source, PropsSchema: r.PropsSchema, Bindings: r.Bindings, Meta: r.Meta, Active: r.Active})
	if e != nil {
		return transport.WriteErrorResponse(c, e)
	}
	out, e := h.response(c, v, p)
	if e != nil {
		return transport.WriteErrorResponse(c, e)
	}
	return c.Status(201).JSON(out)
}

// GetByIdentity godoc
// @Summary Получить компонент
// @Description Возвращает активный компонент по identity в пределах проекта.
// @Tags components
// @Produce json
// @Param project_identity path string true "Project identity" example(demo-project)
// @Param component_identity path string true "Component identity" example(user-card)
// @Success 200 {object} ComponentResponse
// @Failure 400 {object} transport.ErrorResponse
// @Failure 404 {object} transport.ErrorResponse
// @Failure 500 {object} transport.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{project_identity}/components/{component_identity} [get]
func (h *Handler) GetByIdentity(c *fiber.Ctx) error {
	p := c.Params("project_identity")
	v, e := h.service.GetByIdentity(c.UserContext(), adapters.GetComponentInput{ProjectIdentity: p, ComponentIdentity: c.Params("component_identity")})
	if e != nil {
		return transport.WriteErrorResponse(c, e)
	}
	out, e := h.response(c, v, p)
	if e != nil {
		return transport.WriteErrorResponse(c, e)
	}
	return c.JSON(out)
}

// List godoc
// @Summary Список компонентов
// @Description Возвращает неудаленные компоненты проекта с optional фильтрами папки и типа.
// @Tags components
// @Produce json
// @Param project_identity path string true "Project identity" example(demo-project)
// @Param folder_identity query string false "Folder identity" example(root-components)
// @Param component_type query string false "Component type" Enums(component-sfc)
// @Success 200 {object} ComponentsListResponse
// @Failure 400 {object} transport.ErrorResponse
// @Failure 404 {object} transport.ErrorResponse
// @Failure 500 {object} transport.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{project_identity}/components [get]
func (h *Handler) List(c *fiber.Ctx) error {
	p := c.Params("project_identity")
	var folder *string
	if q := c.Query("folder_identity"); q != "" {
		folder = &q
	}
	var typ *entities.ComponentType
	if q := c.Query("component_type"); q != "" {
		v := entities.ComponentType(q)
		typ = &v
	}
	values, e := h.service.List(c.UserContext(), adapters.ListComponentsInput{ProjectIdentity: p, FolderIdentity: folder, ComponentType: typ})
	if e != nil {
		return transport.WriteErrorResponse(c, e)
	}
	items := make([]*ComponentResponse, 0, len(values))
	for _, v := range values {
		out, e := h.response(c, v, p)
		if e != nil {
			return transport.WriteErrorResponse(c, e)
		}
		items = append(items, out)
	}
	return c.JSON(ComponentsListResponse{Items: items})
}

// Update godoc
// @Summary Обновить компонент
// @Description Заменяет editable payload компонента, сохраняя id, identity, createdAt и deletedAt.
// @Tags components
// @Accept json
// @Produce json
// @Param project_identity path string true "Project identity" example(demo-project)
// @Param component_identity path string true "Component identity" example(user-card)
// @Param request body UpdateComponentRequest true "Параметры обновления компонента"
// @Success 200 {object} ComponentResponse
// @Failure 400 {object} transport.ErrorResponse
// @Failure 404 {object} transport.ErrorResponse
// @Failure 500 {object} transport.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{project_identity}/components/{component_identity} [patch]
func (h *Handler) Update(c *fiber.Ctx) error {
	var r UpdateComponentRequest
	if e := c.BodyParser(&r); e != nil {
		return transport.WriteErrorResponse(c, transport.ErrInvalidBody)
	}
	if e := h.validator.Validate(&r); e != nil {
		return transport.WriteErrorResponse(c, transport.ErrValidationError)
	}
	p := c.Params("project_identity")
	v, e := h.service.Update(c.UserContext(), adapters.UpdateComponentInput{ProjectIdentity: p, ComponentIdentity: c.Params("component_identity"), FolderIdentity: r.FolderIdentity, DisplayName: r.DisplayName, Description: r.Description, ComponentType: r.ComponentType, Source: r.Source, PropsSchema: r.PropsSchema, Bindings: r.Bindings, Meta: r.Meta, Active: r.Active})
	if e != nil {
		return transport.WriteErrorResponse(c, e)
	}
	out, e := h.response(c, v, p)
	if e != nil {
		return transport.WriteErrorResponse(c, e)
	}
	return c.JSON(out)
}

// SoftDelete godoc
// @Summary Удалить компонент
// @Description Выполняет soft-delete компонента по identity.
// @Tags components
// @Param project_identity path string true "Project identity"
// @Param component_identity path string true "Component identity"
// @Success 204
// @Failure 400 {object} transport.ErrorResponse
// @Failure 404 {object} transport.ErrorResponse
// @Failure 500 {object} transport.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{project_identity}/components/{component_identity} [delete]
func (h *Handler) SoftDelete(c *fiber.Ctx) error { return h.change(c, h.service.SoftDelete) }

// Restore godoc
// @Summary Восстановить компонент
// @Description Восстанавливает soft-deleted компонент по identity.
// @Tags components
// @Param project_identity path string true "Project identity"
// @Param component_identity path string true "Component identity"
// @Success 204
// @Failure 400 {object} transport.ErrorResponse
// @Failure 404 {object} transport.ErrorResponse
// @Failure 500 {object} transport.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{project_identity}/components/{component_identity}/restore [post]
func (h *Handler) Restore(c *fiber.Ctx) error { return h.change(c, h.service.Restore) }

// HardDelete godoc
// @Summary Физически удалить компонент
// @Description Выполняет hard-delete компонента по identity.
// @Tags components
// @Param project_identity path string true "Project identity"
// @Param component_identity path string true "Component identity"
// @Success 204
// @Failure 400 {object} transport.ErrorResponse
// @Failure 404 {object} transport.ErrorResponse
// @Failure 500 {object} transport.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{project_identity}/components/{component_identity}/hard [delete]
func (h *Handler) HardDelete(c *fiber.Ctx) error { return h.change(c, h.service.HardDelete) }

func (h *Handler) change(c *fiber.Ctx, fn func(context.Context, adapters.ComponentIdentityInput) error) error {
	if err := fn(c.UserContext(), adapters.ComponentIdentityInput{ProjectIdentity: c.Params("project_identity"), ComponentIdentity: c.Params("component_identity")}); err != nil {
		return transport.WriteErrorResponse(c, err)
	}
	return c.SendStatus(204)
}
func (h *Handler) TraceMiddleware(n string) fiber.Handler {
	return servicefiber.TraceMiddleware(h.tracer, h.logger, n)
}
