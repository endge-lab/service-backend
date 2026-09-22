package build_profile

import (
	"github.com/endge-lab/service-backend/internal/api/http/respond"
	"github.com/endge-lab/service-backend/internal/api/http/v1/shared"
	resourceusecase "github.com/endge-lab/service-backend/internal/usecase/build_profiles"
	appvalidator "github.com/endge-lab/service-kit-go/pkg/validator"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	usecase   UseCase
	validator appvalidator.Validator
}

func NewHandler(usecase UseCase, validator appvalidator.Validator) *Handler {
	return &Handler{usecase: usecase, validator: validator}
}

// List возвращает общие и собственные личные профили текущего workspace.
// @Summary Получить профили сборки
// @Description Возвращает общие профили workspace и личные профили текущего actor.
// @ID listBuildProfiles
// @Tags Профили сборки
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства" example(default)
// @Success 200 {object} ListResponse
// @Failure 401 {object} shared.ErrorResponse
// @Failure 403 {object} shared.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/build-profiles [get]
func (h *Handler) List(c *fiber.Ctx) error {
	items, err := h.usecase.List(c.UserContext())
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.JSON(ListResponse{Items: items, Total: len(items)})
}

// Create сохраняет текущие настройки как новый профиль.
// @Summary Создать профиль сборки
// @Description Сохраняет новый личный или общий профиль с автоматически сформированными identity и именем.
// @ID createBuildProfile
// @Tags Профили сборки
// @Accept json
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства" example(default)
// @Param request body CreateRequest true "Профиль"
// @Success 201 {object} Response
// @Failure 400 {object} shared.ErrorResponse
// @Failure 401 {object} shared.ErrorResponse
// @Failure 403 {object} shared.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/build-profiles [post]
func (h *Handler) Create(c *fiber.Ctx) error {
	request, err := shared.DecodeAndValidate[CreateRequest](c, h.validator)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	value, err := h.usecase.Create(c.UserContext(), request.Visibility, request.Settings)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	c.Set(fiber.HeaderETag, shared.ETag(value.Revision))
	return c.Status(fiber.StatusCreated).JSON(value)
}

// Patch изменяет профиль с обязательной проверкой revision.
// @Summary Изменить профиль сборки
// @Description Изменяет имя, видимость или настройки профиля с оптимистической проверкой revision.
// @ID patchBuildProfile
// @Tags Профили сборки
// @Accept json
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства" example(default)
// @Param identity path string true "Identity профиля"
// @Param If-Match header string true "Revision"
// @Param request body PatchRequest true "Изменения"
// @Success 200 {object} Response
// @Failure 400 {object} shared.ErrorResponse
// @Failure 401 {object} shared.ErrorResponse
// @Failure 403 {object} shared.ErrorResponse
// @Failure 404 {object} shared.ErrorResponse
// @Failure 409 {object} shared.ErrorResponse
// @Failure 428 {object} shared.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/build-profiles/{identity} [patch]
func (h *Handler) Patch(c *fiber.Ctx) error {
	expected, err := shared.IfMatch(c)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	request, err := shared.DecodeAndValidate[PatchRequest](c, h.validator)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	value, err := h.usecase.Patch(c.UserContext(), c.Params("identity"), resourceusecase.Patch{
		DisplayName: request.DisplayName, Visibility: request.Visibility, Settings: request.Settings,
	}, expected)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	c.Set(fiber.HeaderETag, shared.ETag(value.Revision))
	return c.JSON(value)
}

// Delete физически удаляет профиль с обязательной проверкой revision.
// @Summary Удалить профиль сборки
// @Description Физически удаляет профиль, которым текущий actor вправе управлять, с проверкой revision.
// @ID deleteBuildProfile
// @Tags Профили сборки
// @Param X-Endge-Workspace header string true "Identity рабочего пространства" example(default)
// @Param identity path string true "Identity профиля"
// @Param If-Match header string true "Revision"
// @Success 204
// @Failure 400 {object} shared.ErrorResponse
// @Failure 401 {object} shared.ErrorResponse
// @Failure 403 {object} shared.ErrorResponse
// @Failure 404 {object} shared.ErrorResponse
// @Failure 409 {object} shared.ErrorResponse
// @Failure 428 {object} shared.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/build-profiles/{identity} [delete]
func (h *Handler) Delete(c *fiber.Ctx) error {
	expected, err := shared.IfMatch(c)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	if err = h.usecase.Delete(c.UserContext(), c.Params("identity"), expected); err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
