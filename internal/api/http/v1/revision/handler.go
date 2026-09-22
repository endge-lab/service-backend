package revision

import (
	"github.com/endge-lab/service-backend/internal/api/http/respond"
	"github.com/endge-lab/service-backend/internal/api/http/v1/shared"
	"github.com/gofiber/fiber/v2"
)

// Handler обслуживает историю ревизий документов.
type Handler struct{ usecase UseCase }

// NewHandler создаёт revision HTTP-обработчик.
func NewHandler(usecase UseCase) *Handler { return &Handler{usecase: usecase} }

// List возвращает revisions выбранного документа.
// @Summary Получить ревизии документа
// @Description Возвращает неизменяемую историю ревизий выбранного документа.
// @ID listDocumentRevisions
// @Tags Ревизии
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства" example(default)
// @Param type path string true "Тип документа"
// @Param identity path string true "Identity документа" maxlength(160)
// @Success 200 {object} ListResponse "История ревизий"
// @Failure 400 {object} shared.ErrorResponse "Некорректный запрос"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Недостаточно прав"
// @Failure 404 {object} shared.ErrorResponse "Ресурс не найден"
// @Failure 500 {object} shared.ErrorResponse "Внутренняя ошибка сервера"
// @Security BearerAuth
// @Router /api/v1/domain/documents/{type}/{identity}/revisions [get]
func (h *Handler) List(c *fiber.Ctx) error {
	values, err := h.usecase.List(c.UserContext(), c.Params("type"), c.Params("identity"))
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	items, err := shared.MapValues(values, NewResponse)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.JSON(fiber.Map{"items": items, "total": len(items)})
}

// Get возвращает конкретную revision документа.
// @Summary Получить ревизию документа
// @Description Возвращает снимок конкретной ревизии.
// @ID getDocumentRevision
// @Tags Ревизии
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства" example(default)
// @Param type path string true "Тип документа"
// @Param identity path string true "Identity документа" maxlength(160)
// @Param revisionId path string true "UUID revision" format(uuid)
// @Success 200 {object} Response "Ревизия"
// @Failure 400 {object} shared.ErrorResponse "Некорректный запрос"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Недостаточно прав"
// @Failure 404 {object} shared.ErrorResponse "Ресурс не найден"
// @Failure 500 {object} shared.ErrorResponse "Внутренняя ошибка сервера"
// @Security BearerAuth
// @Router /api/v1/domain/documents/{type}/{identity}/revisions/{revisionId} [get]
func (h *Handler) Get(c *fiber.Ctx) error {
	value, err := h.usecase.Get(c.UserContext(), c.Params("type"), c.Params("identity"), c.Params("revisionId"))
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	response, err := NewResponse(*value)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.JSON(response)
}

// Restore применяет snapshot revision поверх текущего состояния.
// @Summary Восстановить ревизию документа
// @Description Применяет снимок выбранной ревизии поверх текущего документа и добавляет новую ревизию.
// @ID restoreDocumentRevision
// @Tags Ревизии
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства" example(default)
// @Param type path string true "Тип документа"
// @Param identity path string true "Identity документа" maxlength(160)
// @Param revisionId path string true "UUID revision" format(uuid)
// @Param If-Match header string true "Текущая revision документа" example("3")
// @Success 200 {object} RestoreResponse "Восстановленный документ"
// @Header 200 {string} ETag "Новая revision документа"
// @Failure 400 {object} shared.ErrorResponse "Некорректный запрос"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Недостаточно прав"
// @Failure 404 {object} shared.ErrorResponse "Ресурс не найден"
// @Failure 409 {object} shared.ErrorResponse "Конфликт состояния"
// @Failure 428 {object} shared.ErrorResponse "Требуется If-Match"
// @Failure 500 {object} shared.ErrorResponse "Внутренняя ошибка сервера"
// @Security BearerAuth
// @Router /api/v1/domain/documents/{type}/{identity}/revisions/{revisionId}/restore [post]
func (h *Handler) Restore(c *fiber.Ctx) error {
	expected, err := shared.IfMatch(c)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	value, err := h.usecase.Restore(c.UserContext(), c.Params("type"), c.Params("identity"), c.Params("revisionId"), expected)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	response, err := shared.DocumentMap(*value)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	c.Set(fiber.HeaderETag, shared.ETag(value.Revision))
	return c.JSON(response)
}
