package integration

import (
	"context"

	"github.com/endge-lab/service-backend/internal/api/http/respond"
	"github.com/endge-lab/service-backend/internal/api/http/v1/shared"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	appvalidator "github.com/endge-lab/service-kit-go/pkg/validator"
	"github.com/gofiber/fiber/v2"
)

// Handler обслуживает глобальный catalog интеграций.
type Handler struct {
	usecase   UseCase
	validator appvalidator.Validator
}

// NewHandler создаёт HTTP-обработчик catalog интеграций.
func NewHandler(usecase UseCase, validator appvalidator.Validator) *Handler {
	return &Handler{usecase: usecase, validator: validator}
}

// List возвращает список интеграций.
// @Summary Получить каталог интеграций
// @Description Возвращает глобальный каталог интеграций платформы.
// @ID listIntegrations
// @Tags Интеграции
// @Produce json
// @Param includeDeleted query bool false "Включить мягко удалённые интеграции" default(false)
// @Success 200 {object} ListResponse "Каталог интеграций"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Недостаточно прав"
// @Failure 500 {object} shared.ErrorResponse "Внутренняя ошибка сервера"
// @Security BearerAuth
// @Router /api/v1/integrations [get]
func (h *Handler) List(c *fiber.Ctx) error {
	values, err := h.usecase.List(c.UserContext(), c.QueryBool("includeDeleted", false))
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	items, err := shared.MapValues(values, NewResponse)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.JSON(fiber.Map{"items": items, "total": len(items)})
}

// Create проверяет запрос и создаёт интеграцию.
// @Summary Создать интеграцию
// @Description Добавляет интеграцию в глобальный каталог. Операция доступна администратору платформы.
// @ID createIntegration
// @Tags Интеграции
// @Accept json
// @Produce json
// @Param request body CreateRequest true "Данные интеграции"
// @Success 201 {object} Response "Интеграция создана"
// @Header 201 {string} ETag "Текущая revision"
// @Failure 400 {object} shared.ErrorResponse "Некорректный запрос"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Недостаточно прав"
// @Failure 409 {object} shared.ErrorResponse "Конфликт identity или revision"
// @Failure 500 {object} shared.ErrorResponse "Внутренняя ошибка сервера"
// @Security BearerAuth
// @Router /api/v1/integrations [post]
func (h *Handler) Create(c *fiber.Ctx) error {
	request, err := shared.DecodeAndValidate[CreateRequest](c, h.validator)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	value, err := h.usecase.Create(c.UserContext(), request.Input())
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return h.write(c, fiber.StatusCreated, *value)
}

// Get возвращает интеграцию по identity.
// @Summary Получить интеграцию
// @Description Возвращает интеграцию глобального каталога по identity.
// @ID getIntegration
// @Tags Интеграции
// @Produce json
// @Param identity path string true "Identity интеграции" maxlength(160)
// @Param includeDeleted query bool false "Разрешить получение удалённой интеграции" default(false)
// @Success 200 {object} Response "Интеграция"
// @Header 200 {string} ETag "Текущая revision"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Недостаточно прав"
// @Failure 404 {object} shared.ErrorResponse "Интеграция не найдена"
// @Failure 500 {object} shared.ErrorResponse "Внутренняя ошибка сервера"
// @Security BearerAuth
// @Router /api/v1/integrations/{identity} [get]
func (h *Handler) Get(c *fiber.Ctx) error {
	value, err := h.usecase.Get(c.UserContext(), c.Params("identity"), c.QueryBool("includeDeleted", false))
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return h.write(c, fiber.StatusOK, *value)
}

// Patch проверяет If-Match и изменяет интеграцию.
// @Summary Изменить интеграцию
// @Description Частично изменяет интеграцию глобального каталога.
// @ID patchIntegration
// @Tags Интеграции
// @Accept json
// @Produce json
// @Param identity path string true "Identity интеграции" maxlength(160)
// @Param If-Match header string true "Текущая revision" example("3")
// @Param request body PatchRequest true "Изменяемые поля"
// @Success 200 {object} Response "Интеграция изменена"
// @Header 200 {string} ETag "Новая revision"
// @Failure 400 {object} shared.ErrorResponse "Некорректный запрос"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Недостаточно прав"
// @Failure 404 {object} shared.ErrorResponse "Интеграция не найдена"
// @Failure 409 {object} shared.ErrorResponse "Конфликт identity или revision"
// @Failure 428 {object} shared.ErrorResponse "Требуется заголовок If-Match"
// @Failure 500 {object} shared.ErrorResponse "Внутренняя ошибка сервера"
// @Security BearerAuth
// @Router /api/v1/integrations/{identity} [patch]
func (h *Handler) Patch(c *fiber.Ctx) error {
	expected, err := shared.IfMatch(c)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	request, err := shared.DecodeAndValidate[PatchRequest](c, h.validator)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	input, err := request.Input(c.Body())
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	value, err := h.usecase.Patch(c.UserContext(), c.Params("identity"), input, expected)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return h.write(c, fiber.StatusOK, *value)
}

// Delete выполняет мягкое удаление интеграции.
// @Summary Удалить интеграцию
// @Description Выполняет мягкое удаление интеграции глобального каталога.
// @ID deleteIntegration
// @Tags Интеграции
// @Produce json
// @Param identity path string true "Identity интеграции" maxlength(160)
// @Param If-Match header string true "Текущая revision" example("3")
// @Success 200 {object} Response "Интеграция удалена"
// @Header 200 {string} ETag "Новая revision"
// @Failure 400 {object} shared.ErrorResponse "Некорректный запрос"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Недостаточно прав"
// @Failure 404 {object} shared.ErrorResponse "Интеграция не найдена"
// @Failure 409 {object} shared.ErrorResponse "Конфликт identity или revision"
// @Failure 428 {object} shared.ErrorResponse "Требуется заголовок If-Match"
// @Failure 500 {object} shared.ErrorResponse "Внутренняя ошибка сервера"
// @Security BearerAuth
// @Router /api/v1/integrations/{identity} [delete]
func (h *Handler) Delete(c *fiber.Ctx) error { return h.mutate(c, h.usecase.Delete) }

// Restore восстанавливает мягко удалённую интеграцию.
// @Summary Восстановить интеграцию
// @Description Восстанавливает мягко удалённую интеграцию глобального каталога.
// @ID restoreIntegration
// @Tags Интеграции
// @Produce json
// @Param identity path string true "Identity интеграции" maxlength(160)
// @Param If-Match header string true "Текущая revision" example("3")
// @Success 200 {object} Response "Интеграция восстановлена"
// @Header 200 {string} ETag "Новая revision"
// @Failure 400 {object} shared.ErrorResponse "Некорректный запрос"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Недостаточно прав"
// @Failure 404 {object} shared.ErrorResponse "Интеграция не найдена"
// @Failure 409 {object} shared.ErrorResponse "Конфликт identity или revision"
// @Failure 428 {object} shared.ErrorResponse "Требуется заголовок If-Match"
// @Failure 500 {object} shared.ErrorResponse "Внутренняя ошибка сервера"
// @Security BearerAuth
// @Router /api/v1/integrations/{identity}/restore [post]
func (h *Handler) Restore(c *fiber.Ctx) error { return h.mutate(c, h.usecase.Restore) }

func (h *Handler) mutate(c *fiber.Ctx, operation func(context.Context, string, int) (*entities.Integration, error)) error {
	expected, err := shared.IfMatch(c)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	value, err := operation(c.UserContext(), c.Params("identity"), expected)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return h.write(c, fiber.StatusOK, *value)
}

func (h *Handler) write(c *fiber.Ctx, status int, value entities.Integration) error {
	response, err := NewResponse(value)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	c.Set(fiber.HeaderETag, shared.ETag(value.Revision))
	return c.Status(status).JSON(response)
}
