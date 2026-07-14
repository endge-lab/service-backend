package converter

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
	service   adapters.ConverterService
	folders   adapters.FolderService
	validator appvalidator.Validator
	logger    *zap.Logger
	tracer    trace.Tracer
}

func NewHandler(s *usecase.Service, v appvalidator.Validator, l *zap.Logger, t trace.Tracer) *Handler {
	return &Handler{service: s.Converters, folders: s.Folders, validator: v, logger: l, tracer: t}
}

func (h *Handler) response(c *fiber.Ctx, value *entities.Converter, project string) (*ConverterResponse, error) {
	folder, err := h.folders.GetByID(c.UserContext(), value.FolderID)
	if err != nil {
		return nil, err
	}
	return newConverterResponse(value, project, folder.Identity), nil
}

// Create godoc
// @Summary Создать конвертер
// @Description Создает конвертер с JSON source/config. Source не исполняется.
// @Tags converters
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param project_identity path string true "Project identity" example(demo-project)
// @Param request body CreateConverterRequest true "Параметры конвертера"
// @Success 201 {object} ConverterResponse
// @Failure 400 {object} transport.ErrorResponse
// @Failure 404 {object} transport.ErrorResponse
// @Failure 409 {object} transport.ErrorResponse
// @Failure 500 {object} transport.ErrorResponse
// @Router /api/v1/projects/{project_identity}/converters [post]
func (h *Handler) Create(c *fiber.Ctx) error {
	var request CreateConverterRequest
	if err := c.BodyParser(&request); err != nil {
		return transport.WriteErrorResponse(c, transport.ErrInvalidBody)
	}
	if err := h.validator.Validate(&request); err != nil {
		return transport.WriteErrorResponse(c, transport.ErrValidationError)
	}
	project := c.Params("project_identity")
	value, err := h.service.Create(c.UserContext(), adapters.CreateConverterInput{ProjectIdentity: project, FolderIdentity: request.FolderIdentity, Identity: request.Identity, DisplayName: request.DisplayName, Description: request.Description, ConverterType: request.ConverterType, Source: request.Source, IsSystem: request.IsSystem, Meta: request.Meta, Active: request.Active})
	if err != nil {
		return transport.WriteErrorResponse(c, err)
	}
	response, err := h.response(c, value, project)
	if err != nil {
		return transport.WriteErrorResponse(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(response)
}

// List godoc
// @Summary Список конвертеров
// @Description Возвращает неудаленные конвертеры проекта с optional фильтром папки.
// @Tags converters
// @Security BearerAuth
// @Produce json
// @Param project_identity path string true "Project identity"
// @Param folder_identity query string false "Folder identity"
// @Success 200 {object} ConvertersListResponse
// @Failure 400 {object} transport.ErrorResponse
// @Failure 404 {object} transport.ErrorResponse
// @Failure 500 {object} transport.ErrorResponse
// @Router /api/v1/projects/{project_identity}/converters [get]
func (h *Handler) List(c *fiber.Ctx) error {
	project := c.Params("project_identity")
	var folder *string
	if value := c.Query("folder_identity"); value != "" {
		folder = &value
	}
	values, err := h.service.List(c.UserContext(), adapters.ListConvertersInput{ProjectIdentity: project, FolderIdentity: folder})
	if err != nil {
		return transport.WriteErrorResponse(c, err)
	}
	items := make([]*ConverterResponse, 0, len(values))
	for _, value := range values {
		response, err := h.response(c, value, project)
		if err != nil {
			return transport.WriteErrorResponse(c, err)
		}
		items = append(items, response)
	}
	return c.JSON(ConvertersListResponse{Items: items})
}

// GetByIdentity godoc
// @Summary Получить конвертер
// @Description Возвращает активный конвертер по identity.
// @Tags converters
// @Security BearerAuth
// @Produce json
// @Param project_identity path string true "Project identity"
// @Param converter_identity path string true "Converter identity"
// @Success 200 {object} ConverterResponse
// @Failure 400 {object} transport.ErrorResponse
// @Failure 404 {object} transport.ErrorResponse
// @Failure 500 {object} transport.ErrorResponse
// @Router /api/v1/projects/{project_identity}/converters/{converter_identity} [get]
func (h *Handler) GetByIdentity(c *fiber.Ctx) error {
	project := c.Params("project_identity")
	value, err := h.service.GetByIdentity(c.UserContext(), adapters.GetConverterInput{ProjectIdentity: project, ConverterIdentity: c.Params("converter_identity")})
	if err != nil {
		return transport.WriteErrorResponse(c, err)
	}
	response, err := h.response(c, value, project)
	if err != nil {
		return transport.WriteErrorResponse(c, err)
	}
	return c.JSON(response)
}

// Update godoc
// @Summary Обновить конвертер
// @Description Заменяет editable payload конвертера, сохраняя id, identity, createdAt и deletedAt.
// @Tags converters
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param project_identity path string true "Project identity"
// @Param converter_identity path string true "Converter identity"
// @Param request body UpdateConverterRequest true "Параметры обновления конвертера"
// @Success 200 {object} ConverterResponse
// @Failure 400 {object} transport.ErrorResponse
// @Failure 404 {object} transport.ErrorResponse
// @Failure 500 {object} transport.ErrorResponse
// @Router /api/v1/projects/{project_identity}/converters/{converter_identity} [patch]
func (h *Handler) Update(c *fiber.Ctx) error {
	var request UpdateConverterRequest
	if err := c.BodyParser(&request); err != nil {
		return transport.WriteErrorResponse(c, transport.ErrInvalidBody)
	}
	if err := h.validator.Validate(&request); err != nil {
		return transport.WriteErrorResponse(c, transport.ErrValidationError)
	}
	project := c.Params("project_identity")
	value, err := h.service.Update(c.UserContext(), adapters.UpdateConverterInput{ProjectIdentity: project, ConverterIdentity: c.Params("converter_identity"), FolderIdentity: request.FolderIdentity, DisplayName: request.DisplayName, Description: request.Description, ConverterType: request.ConverterType, Source: request.Source, IsSystem: request.IsSystem, Meta: request.Meta, Active: request.Active})
	if err != nil {
		return transport.WriteErrorResponse(c, err)
	}
	response, err := h.response(c, value, project)
	if err != nil {
		return transport.WriteErrorResponse(c, err)
	}
	return c.JSON(response)
}

// SoftDelete godoc
// @Summary Удалить конвертер
// @Description Выполняет soft-delete конвертера.
// @Tags converters
// @Security BearerAuth
// @Param project_identity path string true "Project identity"
// @Param converter_identity path string true "Converter identity"
// @Success 204
// @Failure 400 {object} transport.ErrorResponse
// @Failure 404 {object} transport.ErrorResponse
// @Failure 500 {object} transport.ErrorResponse
// @Router /api/v1/projects/{project_identity}/converters/{converter_identity} [delete]
func (h *Handler) SoftDelete(c *fiber.Ctx) error { return h.change(c, h.service.SoftDelete) }

// Restore godoc
// @Summary Восстановить конвертер
// @Description Восстанавливает soft-deleted конвертер.
// @Tags converters
// @Security BearerAuth
// @Param project_identity path string true "Project identity"
// @Param converter_identity path string true "Converter identity"
// @Success 204
// @Failure 400 {object} transport.ErrorResponse
// @Failure 404 {object} transport.ErrorResponse
// @Failure 500 {object} transport.ErrorResponse
// @Router /api/v1/projects/{project_identity}/converters/{converter_identity}/restore [post]
func (h *Handler) Restore(c *fiber.Ctx) error { return h.change(c, h.service.Restore) }

// HardDelete godoc
// @Summary Физически удалить конвертер
// @Description Выполняет hard-delete конвертера; system converter удалить нельзя.
// @Tags converters
// @Security BearerAuth
// @Param project_identity path string true "Project identity"
// @Param converter_identity path string true "Converter identity"
// @Success 204
// @Failure 400 {object} transport.ErrorResponse
// @Failure 404 {object} transport.ErrorResponse
// @Failure 409 {object} transport.ErrorResponse
// @Failure 500 {object} transport.ErrorResponse
// @Router /api/v1/projects/{project_identity}/converters/{converter_identity}/hard [delete]
func (h *Handler) HardDelete(c *fiber.Ctx) error { return h.change(c, h.service.HardDelete) }

func (h *Handler) change(c *fiber.Ctx, fn func(context.Context, adapters.ConverterIdentityInput) error) error {
	if err := fn(c.UserContext(), adapters.ConverterIdentityInput{ProjectIdentity: c.Params("project_identity"), ConverterIdentity: c.Params("converter_identity")}); err != nil {
		return transport.WriteErrorResponse(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
func (h *Handler) TraceMiddleware(name string) fiber.Handler {
	return servicefiber.TraceMiddleware(h.tracer, h.logger, name)
}
