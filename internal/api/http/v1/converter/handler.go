package converter

import (
	httpobservability "github.com/endge-lab/service-backend/internal/api/http/observability"
	respond "github.com/endge-lab/service-backend/internal/api/http/respond"
	"github.com/endge-lab/service-backend/internal/observability"
	"github.com/endge-lab/service-backend/internal/usecase/converters"
	appvalidator "github.com/endge-lab/service-kit-go/pkg/validator"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service   UseCase
	validator appvalidator.Validator
	observer  observability.Observer
}

func NewHandler(s UseCase, v appvalidator.Validator, core *observability.Core, metrics *httpobservability.HandlerMetrics) *Handler {
	observer := core.For(observability.LayerHandler, "converter_http_handler").WithRecorder(metrics)
	return &Handler{service: s, validator: v, observer: observer}
}

// Create godoc
// @Summary Создать конвертер
// @Description Создает конвертер с JSON source/config. Source не исполняется.
// @Tags converters
// @Param X-Endge-Workspace header string true "Workspace identity" example(demo-workspace)
// @Security BearerAuth && WorkspaceAuth
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
		return respond.RespondDomainError(c, h.observer.Logger(), err)
	}
	return c.Status(fiber.StatusCreated).JSON(h.response(value, project))
}

// List godoc
// @Summary Список конвертеров
// @Description Возвращает неудаленные конвертеры проекта с optional фильтром папки.
// @Tags converters
// @Param X-Endge-Workspace header string true "Workspace identity" example(demo-workspace)
// @Security BearerAuth && WorkspaceAuth
// @Produce json
// @Param project_identity path string true "Project identity" example(demo-project)
// @Param folder_identity query string false "Folder identity" example(shared-converters)
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
		return respond.RespondDomainError(c, h.observer.Logger(), err)
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
// @Param X-Endge-Workspace header string true "Workspace identity" example(demo-workspace)
// @Security BearerAuth && WorkspaceAuth
// @Produce json
// @Param project_identity path string true "Project identity" example(demo-project)
// @Param converter_identity path string true "Converter identity" example(date-to-string)
// @Success 200 {object} ConverterResponse
// @Failure 400 {object} respond.ErrorResponse
// @Failure 404 {object} respond.ErrorResponse
// @Failure 500 {object} respond.ErrorResponse
// @Router /api/v1/projects/{project_identity}/converters/{converter_identity} [get]
func (h *Handler) GetByIdentity(c *fiber.Ctx) error {
	project := c.Params("project_identity")
	value, err := h.service.GetByIdentity(c.UserContext(), converters.GetConverterInput{ProjectIdentity: project, ConverterIdentity: c.Params("converter_identity")})
	if err != nil {
		return respond.RespondDomainError(c, h.observer.Logger(), err)
	}
	return c.JSON(h.response(value, project))
}

// Update godoc
// @Summary Обновить конвертер
// @Description Заменяет editable payload конвертера, сохраняя id, identity, createdAt и deletedAt.
// @Tags converters
// @Param X-Endge-Workspace header string true "Workspace identity" example(demo-workspace)
// @Security BearerAuth && WorkspaceAuth
// @Accept json
// @Produce json
// @Param project_identity path string true "Project identity" example(demo-project)
// @Param converter_identity path string true "Converter identity" example(date-to-string)
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
		return respond.RespondDomainError(c, h.observer.Logger(), err)
	}
	return c.JSON(h.response(value, project))
}

// SoftDelete godoc
// @Summary Удалить конвертер
// @Description Выполняет soft-delete конвертера.
// @Tags converters
// @Param X-Endge-Workspace header string true "Workspace identity" example(demo-workspace)
// @Security BearerAuth && WorkspaceAuth
// @Param project_identity path string true "Project identity" example(demo-project)
// @Param converter_identity path string true "Converter identity" example(date-to-string)
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
// @Param X-Endge-Workspace header string true "Workspace identity" example(demo-workspace)
// @Security BearerAuth && WorkspaceAuth
// @Param project_identity path string true "Project identity" example(demo-project)
// @Param converter_identity path string true "Converter identity" example(restore-date-to-string)
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
// @Param X-Endge-Workspace header string true "Workspace identity" example(demo-workspace)
// @Security BearerAuth && WorkspaceAuth
// @Param project_identity path string true "Project identity" example(demo-project)
// @Param converter_identity path string true "Converter identity" example(hard-delete-date-to-string)
// @Success 204
// @Failure 400 {object} respond.ErrorResponse
// @Failure 404 {object} respond.ErrorResponse
// @Failure 409 {object} respond.ErrorResponse
// @Failure 500 {object} respond.ErrorResponse
// @Router /api/v1/projects/{project_identity}/converters/{converter_identity}/hard [delete]
func (h *Handler) HardDelete(c *fiber.Ctx) error { return h.change(c, h.service.HardDelete) }
