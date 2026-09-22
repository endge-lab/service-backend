package backend_connection

import (
	"github.com/endge-lab/service-backend/internal/api/http/respond"
	"github.com/endge-lab/service-backend/internal/api/http/v1/shared"
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

// List возвращает центральный каталог backend-подключений.
// @Summary Получить backend-подключения
// @Description Возвращает глобальный каталог основного backend без привязки к Workspace.
// @ID listBackendConnections
// @Tags Backend-подключения
// @Produce json
// @Success 200 {object} ListResponse
// @Failure 401 {object} shared.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/backend-connections [get]
func (h *Handler) List(c *fiber.Ctx) error {
	result, err := h.usecase.List(c.UserContext())
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	items := make([]Response, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, newResponse(item))
	}
	return c.JSON(ListResponse{Items: items, Total: len(items), CanManage: result.CanManage})
}

// Create добавляет backend-подключение. Доступно только Platform Admin.
// @Summary Добавить backend-подключение
// @Description Нормализует и добавляет адрес в глобальный каталог основного backend. Доступно только Platform Admin.
// @ID createBackendConnection
// @Tags Backend-подключения
// @Accept json
// @Produce json
// @Param request body CreateRequest true "Название и адрес backend"
// @Success 201 {object} Response
// @Failure 400 {object} shared.ErrorResponse
// @Failure 401 {object} shared.ErrorResponse
// @Failure 403 {object} shared.ErrorResponse
// @Failure 409 {object} shared.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/backend-connections [post]
func (h *Handler) Create(c *fiber.Ctx) error {
	request, err := shared.DecodeAndValidate[CreateRequest](c, h.validator)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	created, err := h.usecase.Create(c.UserContext(), request.Name, request.BaseURL)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.Status(fiber.StatusCreated).JSON(newResponse(*created))
}

// Delete физически удаляет backend-подключение. Доступно только Platform Admin.
// @Summary Удалить backend-подключение
// @Description Физически удаляет адрес из глобального каталога. Доступно только Platform Admin.
// @ID deleteBackendConnection
// @Tags Backend-подключения
// @Param id path string true "ID подключения" format(uuid)
// @Success 204
// @Failure 401 {object} shared.ErrorResponse
// @Failure 403 {object} shared.ErrorResponse
// @Failure 404 {object} shared.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/backend-connections/{id} [delete]
func (h *Handler) Delete(c *fiber.Ctx) error {
	if err := h.usecase.Delete(c.UserContext(), c.Params("id")); err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
