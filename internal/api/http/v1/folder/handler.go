package folder

import (
	"github.com/endge-lab/service-backend/internal/api/http/v1/shared"
	appvalidator "github.com/endge-lab/service-kit-go/pkg/validator"
	"github.com/gofiber/fiber/v2"
)

// Handler обслуживает HTTP-операции ресурса и зависит только от application-порта.
type Handler struct {
	usecase   UseCase
	validator appvalidator.Validator
}

// NewHandler создаёт HTTP-обработчик ресурса.
func NewHandler(usecase UseCase, validator appvalidator.Validator) *Handler {
	return &Handler{usecase: usecase, validator: validator}
}

// List возвращает список документов ресурса с фильтрацией и пагинацией.
// @Summary Получить список папок
// @Description Возвращает список папок текущего рабочего пространства с фильтрацией и пагинацией.
// @ID listFolders
// @Tags Папки
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства" example(default)
// @Param includeDeleted query bool false "Включить мягко удалённые документы" default(false)
// @Param folderIdentity query string false "Identity папки" maxlength(160)
// @Param active query bool false "Фильтр по активности"
// @Param limit query int false "Размер страницы" default(100) minimum(1) maximum(500)
// @Param offset query int false "Смещение" default(0) minimum(0)
// @Success 200 {object} ListResponse "Список папок"
// @Failure 400 {object} shared.ErrorResponse "Некорректный запрос"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Недостаточно прав"
// @Failure 500 {object} shared.ErrorResponse "Внутренняя ошибка сервера"
// @Security BearerAuth
// @Router /api/v1/folders [get]
func (h *Handler) List(c *fiber.Ctx) error {
	return shared.ListDocuments(c, h.usecase.List, NewResponse)
}

// Create проверяет запрос и создаёт документ ресурса.
// @Summary Создать папку
// @Description Создаёт папку в текущем рабочем пространстве.
// @ID createFolder
// @Tags Папки
// @Accept json
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства" example(default)
// @Param request body CreateRequest true "Данные нового документа"
// @Success 201 {object} Response "Документ создан"
// @Header 201 {string} ETag "Текущая revision документа"
// @Failure 400 {object} shared.ErrorResponse "Некорректный запрос"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Недостаточно прав"
// @Failure 409 {object} shared.ErrorResponse "Конфликт identity или revision"
// @Failure 500 {object} shared.ErrorResponse "Внутренняя ошибка сервера"
// @Security BearerAuth
// @Router /api/v1/folders [post]
func (h *Handler) Create(c *fiber.Ctx) error {
	return shared.CreateDocument[CreateRequest](c, h.validator, h.usecase.Create, NewResponse)
}

// Get возвращает документ ресурса по identity.
// @Summary Получить папку
// @Description Возвращает папку по identity.
// @ID getFolder
// @Tags Папки
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства" example(default)
// @Param identity path string true "Identity документа" maxlength(160)
// @Param includeDeleted query bool false "Разрешить получение мягко удалённого документа" default(false)
// @Success 200 {object} Response "Найденный документ"
// @Header 200 {string} ETag "Текущая revision документа"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Недостаточно прав"
// @Failure 404 {object} shared.ErrorResponse "Документ не найден"
// @Failure 500 {object} shared.ErrorResponse "Внутренняя ошибка сервера"
// @Security BearerAuth
// @Router /api/v1/folders/{identity} [get]
func (h *Handler) Get(c *fiber.Ctx) error {
	return shared.GetDocument(c, h.usecase.Get, NewResponse)
}

// Patch проверяет If-Match и частично изменяет документ ресурса.
// @Summary Изменить папку
// @Description Частично изменяет папку; актуальная revision передаётся в If-Match.
// @ID patchFolder
// @Tags Папки
// @Accept json
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства" example(default)
// @Param identity path string true "Identity документа" maxlength(160)
// @Param If-Match header string true "Текущая revision документа" example("3")
// @Param request body PatchRequest true "Изменяемые поля документа"
// @Success 200 {object} Response "Документ изменён"
// @Header 200 {string} ETag "Новая revision документа"
// @Failure 400 {object} shared.ErrorResponse "Некорректный запрос"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Недостаточно прав"
// @Failure 404 {object} shared.ErrorResponse "Документ не найден"
// @Failure 409 {object} shared.ErrorResponse "Конфликт identity или revision"
// @Failure 428 {object} shared.ErrorResponse "Требуется заголовок If-Match"
// @Failure 500 {object} shared.ErrorResponse "Внутренняя ошибка сервера"
// @Security BearerAuth
// @Router /api/v1/folders/{identity} [patch]
func (h *Handler) Patch(c *fiber.Ctx) error {
	return shared.PatchDocument[PatchRequest](c, h.validator, h.usecase.Patch, NewResponse)
}

// Delete выполняет мягкое удаление документа ресурса.
// @Summary Удалить папку
// @Description Выполняет мягкое удаление документа; актуальная revision передаётся в If-Match.
// @ID deleteFolder
// @Tags Папки
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства" example(default)
// @Param identity path string true "Identity документа" maxlength(160)
// @Param If-Match header string true "Текущая revision документа" example("3")
// @Success 200 {object} Response "Документ мягко удалён"
// @Header 200 {string} ETag "Новая revision документа"
// @Failure 400 {object} shared.ErrorResponse "Некорректный запрос"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Недостаточно прав"
// @Failure 404 {object} shared.ErrorResponse "Документ не найден"
// @Failure 409 {object} shared.ErrorResponse "Конфликт identity или revision"
// @Failure 428 {object} shared.ErrorResponse "Требуется заголовок If-Match"
// @Failure 500 {object} shared.ErrorResponse "Внутренняя ошибка сервера"
// @Security BearerAuth
// @Router /api/v1/folders/{identity} [delete]
func (h *Handler) Delete(c *fiber.Ctx) error {
	return shared.MutateDocument(c, h.usecase.Delete, NewResponse)
}

// Restore восстанавливает мягко удалённый документ ресурса.
// @Summary Восстановить папку
// @Description Восстанавливает мягко удалённый документ; актуальная revision передаётся в If-Match.
// @ID restoreFolder
// @Tags Папки
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства" example(default)
// @Param identity path string true "Identity документа" maxlength(160)
// @Param If-Match header string true "Текущая revision документа" example("3")
// @Success 200 {object} Response "Документ восстановлен"
// @Header 200 {string} ETag "Новая revision документа"
// @Failure 400 {object} shared.ErrorResponse "Некорректный запрос"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Недостаточно прав"
// @Failure 404 {object} shared.ErrorResponse "Документ не найден"
// @Failure 409 {object} shared.ErrorResponse "Конфликт identity или revision"
// @Failure 428 {object} shared.ErrorResponse "Требуется заголовок If-Match"
// @Failure 500 {object} shared.ErrorResponse "Внутренняя ошибка сервера"
// @Security BearerAuth
// @Router /api/v1/folders/{identity}/restore [post]
func (h *Handler) Restore(c *fiber.Ctx) error {
	return shared.MutateDocument(c, h.usecase.Restore, NewResponse)
}
