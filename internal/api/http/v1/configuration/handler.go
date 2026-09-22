package configuration

import (
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

// List godoc
// @Summary Получить список Configuration-документов
// @Description Возвращает Configuration-документы текущего рабочего пространства.
// @ID listConfigurations
// @Tags Конфигурации
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства" example(default)
// @Param includeDeleted query bool false "Включить мягко удалённые документы" default(false)
// @Param active query bool false "Фильтр по активности"
// @Param limit query int false "Размер страницы" default(100) minimum(1) maximum(500)
// @Param offset query int false "Смещение" default(0) minimum(0)
// @Success 200 {object} ListResponse
// @Security BearerAuth
// @Router /api/v1/configurations [get]
func (h *Handler) List(c *fiber.Ctx) error {
	return shared.ListDocuments(c, h.usecase.List, NewResponse)
}

// Create godoc
// @Summary Создать Configuration-документ
// @Description Создаёт Configuration sourceVersion 1 без папки.
// @ID createConfiguration
// @Tags Конфигурации
// @Accept json
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства" example(default)
// @Param request body CreateRequest true "Документ"
// @Success 201 {object} Response
// @Header 201 {string} ETag "Текущая revision документа"
// @Security BearerAuth
// @Router /api/v1/configurations [post]
func (h *Handler) Create(c *fiber.Ctx) error {
	return shared.CreateDocument[CreateRequest](c, h.validator, h.usecase.Create, NewResponse)
}

// Get godoc
// @Summary Получить Configuration-документ
// @Description Возвращает Configuration-документ по identity.
// @ID getConfiguration
// @Tags Конфигурации
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства" example(default)
// @Param identity path string true "Identity"
// @Param includeDeleted query bool false "Разрешить получение мягко удалённого документа" default(false)
// @Success 200 {object} Response
// @Header 200 {string} ETag "Текущая revision документа"
// @Security BearerAuth
// @Router /api/v1/configurations/{identity} [get]
func (h *Handler) Get(c *fiber.Ctx) error {
	return shared.GetDocument(c, h.usecase.Get, NewResponse)
}

// Patch godoc
// @Summary Изменить Configuration-документ
// @Description Частично изменяет Configuration с optimistic revision check.
// @ID patchConfiguration
// @Tags Конфигурации
// @Accept json
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства" example(default)
// @Param identity path string true "Identity"
// @Param If-Match header string true "Текущая revision документа" example("3")
// @Param request body PatchRequest true "Изменения"
// @Success 200 {object} Response
// @Header 200 {string} ETag "Новая revision документа"
// @Failure 409 {object} shared.ErrorResponse "Конфликт identity или revision"
// @Failure 428 {object} shared.ErrorResponse "Требуется заголовок If-Match"
// @Security BearerAuth
// @Router /api/v1/configurations/{identity} [patch]
func (h *Handler) Patch(c *fiber.Ctx) error {
	return shared.PatchDocument[PatchRequest](c, h.validator, h.usecase.Patch, NewResponse)
}

// Delete godoc
// @Summary Удалить Configuration-документ
// @Description Выполняет мягкое удаление Configuration-документа.
// @ID deleteConfiguration
// @Tags Конфигурации
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства" example(default)
// @Param identity path string true "Identity"
// @Param If-Match header string true "Текущая revision документа" example("3")
// @Success 200 {object} Response
// @Header 200 {string} ETag "Новая revision документа"
// @Failure 409 {object} shared.ErrorResponse "Конфликт identity или revision"
// @Failure 428 {object} shared.ErrorResponse "Требуется заголовок If-Match"
// @Security BearerAuth
// @Router /api/v1/configurations/{identity} [delete]
func (h *Handler) Delete(c *fiber.Ctx) error {
	return shared.MutateDocument(c, h.usecase.Delete, NewResponse)
}

// Restore godoc
// @Summary Восстановить Configuration-документ
// @Description Восстанавливает мягко удалённый Configuration-документ.
// @ID restoreConfiguration
// @Tags Конфигурации
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства" example(default)
// @Param identity path string true "Identity"
// @Param If-Match header string true "Текущая revision документа" example("3")
// @Success 200 {object} Response
// @Header 200 {string} ETag "Новая revision документа"
// @Failure 409 {object} shared.ErrorResponse "Конфликт identity или revision"
// @Failure 428 {object} shared.ErrorResponse "Требуется заголовок If-Match"
// @Security BearerAuth
// @Router /api/v1/configurations/{identity}/restore [post]
func (h *Handler) Restore(c *fiber.Ctx) error {
	return shared.MutateDocument(c, h.usecase.Restore, NewResponse)
}
