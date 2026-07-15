package folder

import (
	transport "github.com/endge-lab/service-backend/internal/api/http/v1/transport"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase"
	"github.com/endge-lab/service-backend/internal/usecase/adapters"
	servicefiber "github.com/endge-lab/service-kit-go/pkg/httpkit/fiber"
	"github.com/endge-lab/service-kit-go/pkg/logging"
	appvalidator "github.com/endge-lab/service-kit-go/pkg/validator"
	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type ErrorResponse = transport.ErrorResponse

type Handler struct {
	folderService adapters.FolderService
	validator     appvalidator.Validator
	logger        *zap.Logger
	tracer        trace.Tracer
}

func NewHandler(
	service *usecase.Service,
	validator appvalidator.Validator,
	logger *zap.Logger,
	tracer trace.Tracer,
) *Handler {
	return &Handler{
		folderService: service.Folders,
		validator:     validator,
		logger:        logger.With(zap.String("component", "folder_http_handler")),
		tracer:        tracer,
	}
}

// CreateFolder godoc
// @Summary Создать папку
// @Description Создает папку внутри проекта и дерева указанного entity type.
// @Tags folders
// @Accept json
// @Produce json
// @Param project_identity path string true "Project identity" example(demo-project)
// @Param request body CreateFolderRequest true "Параметры папки"
// @Success 201 {object} FolderResponse
// @Failure 400 {object} transport.ErrorResponse
// @Failure 404 {object} transport.ErrorResponse
// @Failure 409 {object} transport.ErrorResponse
// @Failure 500 {object} transport.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{project_identity}/folders [post]
func (h *Handler) CreateFolder(c *fiber.Ctx) error {
	var request CreateFolderRequest
	if err := c.BodyParser(&request); err != nil {
		return transport.WriteErrorResponse(c, transport.ErrInvalidBody)
	}
	if err := h.validator.Validate(&request); err != nil {
		return transport.WriteErrorResponse(c, transport.ErrValidationError)
	}

	projectIdentity := c.Params("project_identity")
	folder, err := h.folderService.Create(c.UserContext(), adapters.CreateFolderInput{
		ProjectIdentity: projectIdentity,
		EntityType:      request.EntityType,
		Identity:        request.Identity,
		DisplayName:     request.DisplayName,
		Description:     request.Description,
		ParentIdentity:  request.ParentIdentity,
		Meta:            request.Meta,
	})
	if err != nil {
		return h.respondDomainError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(
		NewFolderResponse(folder, projectIdentity, request.ParentIdentity),
	)
}

// ListFolders godoc
// @Summary Список папок
// @Description Возвращает неудаленные папки проекта для указанного entity type.
// @Tags folders
// @Produce json
// @Param project_identity path string true "Project identity" example(demo-project)
// @Param entity_type query string true "Entity type" Enums(components,converters,queries,data-views) example(components)
// @Success 200 {object} FoldersListResponse
// @Failure 400 {object} transport.ErrorResponse
// @Failure 404 {object} transport.ErrorResponse
// @Failure 500 {object} transport.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{project_identity}/folders [get]
func (h *Handler) ListFolders(c *fiber.Ctx) error {
	projectIdentity := c.Params("project_identity")
	entityType := entities.FolderEntityType(c.Query("entity_type"))

	folders, err := h.folderService.List(c.UserContext(), adapters.ListFoldersInput{
		ProjectIdentity: projectIdentity,
		EntityType:      entityType,
	})
	if err != nil {
		return h.respondDomainError(c, err)
	}

	items := make([]*FolderResponse, 0, len(folders))
	for _, item := range folders {
		response, err := h.newFolderResponse(c, projectIdentity, item)
		if err != nil {
			return h.respondDomainError(c, err)
		}
		items = append(items, response)
	}

	return c.Status(fiber.StatusOK).JSON(FoldersListResponse{Items: items})
}

// GetFolderByIdentity godoc
// @Summary Получить папку
// @Description Возвращает папку по project identity, folder identity и entity type.
// @Tags folders
// @Produce json
// @Param project_identity path string true "Project identity" example(demo-project)
// @Param folder_identity path string true "Folder identity" example(shared-components)
// @Param entity_type query string true "Entity type" Enums(components,converters,queries,data-views) example(components)
// @Success 200 {object} FolderResponse
// @Failure 400 {object} transport.ErrorResponse
// @Failure 404 {object} transport.ErrorResponse
// @Failure 500 {object} transport.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{project_identity}/folders/{folder_identity} [get]
func (h *Handler) GetFolderByIdentity(c *fiber.Ctx) error {
	input := h.folderIdentityInput(c)
	folder, err := h.folderService.GetByIdentity(c.UserContext(), adapters.GetFolderInput(input))
	if err != nil {
		return h.respondDomainError(c, err)
	}

	response, err := h.newFolderResponse(c, input.ProjectIdentity, folder)
	if err != nil {
		return h.respondDomainError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

// UpdateFolder godoc
// @Summary Обновить папку
// @Description Обновляет папку и при необходимости перемещает ее по parent identity.
// @Tags folders
// @Accept json
// @Produce json
// @Param project_identity path string true "Project identity" example(demo-project)
// @Param folder_identity path string true "Folder identity" example(shared-components)
// @Param entity_type query string true "Entity type" Enums(components,converters,queries,data-views) example(components)
// @Param request body UpdateFolderRequest true "Параметры обновления папки"
// @Success 200 {object} FolderResponse
// @Failure 400 {object} transport.ErrorResponse
// @Failure 404 {object} transport.ErrorResponse
// @Failure 409 {object} transport.ErrorResponse
// @Failure 500 {object} transport.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{project_identity}/folders/{folder_identity} [patch]
func (h *Handler) UpdateFolder(c *fiber.Ctx) error {
	var request UpdateFolderRequest
	if err := c.BodyParser(&request); err != nil {
		return transport.WriteErrorResponse(c, transport.ErrInvalidBody)
	}
	if err := h.validator.Validate(&request); err != nil {
		return transport.WriteErrorResponse(c, transport.ErrValidationError)
	}

	input := h.folderIdentityInput(c)
	folder, err := h.folderService.Update(c.UserContext(), adapters.UpdateFolderInput{
		ProjectIdentity: input.ProjectIdentity,
		EntityType:      input.EntityType,
		Identity:        input.Identity,
		DisplayName:     request.DisplayName,
		Description:     request.Description,
		ParentIdentity:  request.ParentIdentity,
		Meta:            request.Meta,
	})
	if err != nil {
		return h.respondDomainError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(
		NewFolderResponse(folder, input.ProjectIdentity, request.ParentIdentity),
	)
}

// SoftDeleteFolder godoc
// @Summary Удалить папку
// @Description Выполняет soft delete папки.
// @Tags folders
// @Param project_identity path string true "Project identity" example(demo-project)
// @Param folder_identity path string true "Folder identity" example(shared-components)
// @Param entity_type query string true "Entity type" Enums(components,converters,queries,data-views) example(components)
// @Success 204
// @Failure 400 {object} transport.ErrorResponse
// @Failure 404 {object} transport.ErrorResponse
// @Failure 500 {object} transport.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{project_identity}/folders/{folder_identity} [delete]
func (h *Handler) SoftDeleteFolder(c *fiber.Ctx) error {
	if err := h.folderService.SoftDelete(c.UserContext(), h.folderIdentityInput(c)); err != nil {
		return h.respondDomainError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// RestoreFolder godoc
// @Summary Восстановить папку
// @Description Восстанавливает soft-deleted папку.
// @Tags folders
// @Param project_identity path string true "Project identity" example(demo-project)
// @Param folder_identity path string true "Folder identity" example(shared-components)
// @Param entity_type query string true "Entity type" Enums(components,converters,queries,data-views) example(components)
// @Success 204
// @Failure 400 {object} transport.ErrorResponse
// @Failure 404 {object} transport.ErrorResponse
// @Failure 500 {object} transport.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{project_identity}/folders/{folder_identity}/restore [post]
func (h *Handler) RestoreFolder(c *fiber.Ctx) error {
	if err := h.folderService.Restore(c.UserContext(), h.folderIdentityInput(c)); err != nil {
		return h.respondDomainError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// HardDeleteFolder godoc
// @Summary Удалить папку физически
// @Description Выполняет hard delete папки, кроме system root folders.
// @Tags folders
// @Param project_identity path string true "Project identity" example(demo-project)
// @Param folder_identity path string true "Folder identity" example(shared-components)
// @Param entity_type query string true "Entity type" Enums(components,converters,queries,data-views) example(components)
// @Success 204
// @Failure 400 {object} transport.ErrorResponse
// @Failure 404 {object} transport.ErrorResponse
// @Failure 409 {object} transport.ErrorResponse
// @Failure 500 {object} transport.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/projects/{project_identity}/folders/{folder_identity}/hard [delete]
func (h *Handler) HardDeleteFolder(c *fiber.Ctx) error {
	if err := h.folderService.HardDelete(c.UserContext(), h.folderIdentityInput(c)); err != nil {
		return h.respondDomainError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) folderIdentityInput(c *fiber.Ctx) adapters.FolderIdentityInput {
	return adapters.FolderIdentityInput{
		ProjectIdentity: c.Params("project_identity"),
		EntityType:      entities.FolderEntityType(c.Query("entity_type")),
		Identity:        c.Params("folder_identity"),
	}
}

func (h *Handler) newFolderResponse(
	c *fiber.Ctx,
	projectIdentity string,
	folder *entities.Folder,
) (*FolderResponse, error) {
	if folder == nil || folder.ParentID == nil {
		return NewFolderResponse(folder, projectIdentity, nil), nil
	}

	parent, err := h.folderService.GetByID(c.UserContext(), *folder.ParentID)
	if err != nil {
		return nil, err
	}

	parentIdentity := parent.Identity
	return NewFolderResponse(folder, projectIdentity, &parentIdentity), nil
}

func (h *Handler) TraceMiddleware(spanName string) fiber.Handler {
	return servicefiber.TraceMiddleware(h.tracer, h.logger, spanName)
}

func (h *Handler) respondDomainError(c *fiber.Ctx, err error) error {
	fields := []zap.Field{
		zap.Error(err),
		zap.String("error_code", domainerrors.CodeOf(err)),
		zap.String("method", c.Method()),
		zap.String("path", c.Path()),
	}

	logger := logging.WithContext(c.UserContext(), h.logger)
	if domainerrors.HTTPStatusOf(err) >= fiber.StatusInternalServerError {
		logger.Error("unexpected request transport", fields...)
	} else {
		logger.Warn("request completed with business transport", fields...)
	}

	return transport.WriteErrorResponse(c, err)
}
