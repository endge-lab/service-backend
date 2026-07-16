package converter

import (
	respond "github.com/endge-lab/service-backend/internal/api/http/respond"
	"github.com/endge-lab/service-backend/internal/usecase/converters"
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
	return &Handler{service: s, validator: v, logger: l.With(zap.String("component", "converter_http_handler")), tracer: t}
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
// @Failure 400 {object} respond.ErrorResponse
// @Failure 404 {object} respond.ErrorResponse
// @Failure 409 {object} respond.ErrorResponse
// @Failure 500 {object} respond.ErrorResponse
// @Router /api/v1/projects/{project_identity}/converters [post]
func (h *Handler) Create(c *fiber.Ctx) error {
	var request CreateConverterRequest
	if err := c.BodyParser(&request); err != nil {
		return respond.WriteErrorResponse(c, respond.ErrInvalidBody)
	}
	if err := h.validator.Validate(&request); err != nil {
		return respond.WriteErrorResponse(c, respond.ErrValidationError)
	}
	project := c.Params("project_identity")
	value, err := h.service.Create(c.UserContext(), converters.CreateConverterInput{ProjectIdentity: project, FolderIdentity: request.FolderIdentity, Identity: request.Identity, DisplayName: request.DisplayName, Description: request.Description, ConverterType: request.ConverterType, Source: request.Source, IsSystem: request.IsSystem, Meta: request.Meta, Active: request.Active})
	if err != nil {
		return respond.RespondDomainError(c, h.logger, err)
	}
	return c.Status(fiber.StatusCreated).JSON(h.response(value, project))
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
// @Failure 400 {object} respond.ErrorResponse
// @Failure 404 {object} respond.ErrorResponse
// @Failure 500 {object} respond.ErrorResponse
// @Router /api/v1/projects/{project_identity}/converters [get]
func (h *Handler) List(c *fiber.Ctx) error {
	project := c.Params("project_identity")
	var folder *string
	if value := c.Query("folder_identity"); value != "" {
		folder = &value
	}
	values, err := h.service.List(c.UserContext(), converters.ListConvertersInput{ProjectIdentity: project, FolderIdentity: folder})
	if err != nil {
		return respond.RespondDomainError(c, h.logger, err)
	}
	items := make([]*ConverterResponse, 0, len(values))
	for _, value := range values {
		items = append(items, h.response(value, project))
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
// @Failure 400 {object} respond.ErrorResponse
// @Failure 404 {object} respond.ErrorResponse
// @Failure 500 {object} respond.ErrorResponse
// @Router /api/v1/projects/{project_identity}/converters/{converter_identity} [get]
func (h *Handler) GetByIdentity(c *fiber.Ctx) error {
	project := c.Params("project_identity")
	value, err := h.service.GetByIdentity(c.UserContext(), converters.GetConverterInput{ProjectIdentity: project, ConverterIdentity: c.Params("converter_identity")})
	if err != nil {
		return respond.RespondDomainError(c, h.logger, err)
	}
	return c.JSON(h.response(value, project))
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
// @Failure 400 {object} respond.ErrorResponse
// @Failure 404 {object} respond.ErrorResponse
// @Failure 500 {object} respond.ErrorResponse
// @Router /api/v1/projects/{project_identity}/converters/{converter_identity} [patch]
func (h *Handler) Update(c *fiber.Ctx) error {
	var request UpdateConverterRequest
	if err := c.BodyParser(&request); err != nil {
		return respond.WriteErrorResponse(c, respond.ErrInvalidBody)
	}
	if err := h.validator.Validate(&request); err != nil {
		return respond.WriteErrorResponse(c, respond.ErrValidationError)
	}
	project := c.Params("project_identity")
	value, err := h.service.Update(c.UserContext(), converters.UpdateConverterInput{ProjectIdentity: project, ConverterIdentity: c.Params("converter_identity"), FolderIdentity: request.FolderIdentity, DisplayName: request.DisplayName, Description: request.Description, ConverterType: request.ConverterType, Source: request.Source, IsSystem: request.IsSystem, Meta: request.Meta, Active: request.Active})
	if err != nil {
		return respond.RespondDomainError(c, h.logger, err)
	}
	return c.JSON(h.response(value, project))
}

// SoftDelete godoc
// @Summary Удалить конвертер
// @Description Выполняет soft-delete конвертера.
// @Tags converters
// @Security BearerAuth
// @Param project_identity path string true "Project identity"
// @Param converter_identity path string true "Converter identity"
// @Success 204
// @Failure 400 {object} respond.ErrorResponse
// @Failure 404 {object} respond.ErrorResponse
// @Failure 500 {object} respond.ErrorResponse
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
// @Failure 400 {object} respond.ErrorResponse
// @Failure 404 {object} respond.ErrorResponse
// @Failure 500 {object} respond.ErrorResponse
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
// @Failure 400 {object} respond.ErrorResponse
// @Failure 404 {object} respond.ErrorResponse
// @Failure 409 {object} respond.ErrorResponse
// @Failure 500 {object} respond.ErrorResponse
// @Router /api/v1/projects/{project_identity}/converters/{converter_identity}/hard [delete]
func (h *Handler) HardDelete(c *fiber.Ctx) error { return h.change(c, h.service.HardDelete) }
