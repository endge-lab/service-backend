package commit

import (
	"github.com/endge-lab/service-backend/internal/api/http/respond"
	"github.com/endge-lab/service-backend/internal/api/http/v1/shared"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	appvalidator "github.com/endge-lab/service-kit-go/pkg/validator"
	"github.com/gofiber/fiber/v2"
)

// Handler обслуживает workspace commit HTTP-операции.
type Handler struct {
	usecase   UseCase
	validator appvalidator.Validator
}

// NewHandler создаёт commit HTTP-обработчик.
func NewHandler(usecase UseCase, validator appvalidator.Validator) *Handler {
	return &Handler{usecase: usecase, validator: validator}
}

// Plan возвращает preview ожидающих commit изменений.
// @Summary Предварительно рассчитать коммит
// @Description Возвращает ревизии, ожидающие фиксации, и их авторов без изменения состояния.
// @ID planCommit
// @Tags Коммиты
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства" example(default)
// @Success 200 {object} PlanResponse "План коммита"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Недостаточно прав"
// @Failure 500 {object} shared.ErrorResponse "Внутренняя ошибка сервера"
// @Security BearerAuth
// @Router /api/v1/commits/plan [post]
func (h *Handler) Plan(c *fiber.Ctx) error {
	value, err := h.usecase.Plan(c.UserContext())
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.JSON(value)
}

// Create фиксирует ожидающие revisions в неизменяемом commit.
// @Summary Создать коммит
// @Description Фиксирует ревизии рабочего пространства, ожидающие фиксации с политикой preserve или squash.
// @ID createCommit
// @Tags Коммиты
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства" example(default)
// @Accept json
// @Param request body CreateRequest true "Параметры commit"
// @Success 201 {object} Response "Коммит создан"
// @Failure 400 {object} shared.ErrorResponse "Некорректный запрос"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Недостаточно прав"
// @Failure 409 {object} shared.ErrorResponse "Конфликт состояния"
// @Failure 500 {object} shared.ErrorResponse "Внутренняя ошибка сервера"
// @Security BearerAuth
// @Router /api/v1/commits [post]
func (h *Handler) Create(c *fiber.Ctx) error {
	request, err := shared.DecodeAndValidate[CreateRequest](c, h.validator)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	value, err := h.usecase.Create(c.UserContext(), request.Message, request.RevisionPolicy, *request.ExpectedHeadSequence)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return h.write(c, fiber.StatusCreated, *value)
}

// List возвращает историю commits рабочего пространства.
// @Summary Получить историю коммитов
// @Description Возвращает коммиты текущего рабочего пространства.
// @ID listCommits
// @Tags Коммиты
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства" example(default)
// @Success 200 {object} ListResponse "История коммитов"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Недостаточно прав"
// @Failure 500 {object} shared.ErrorResponse "Внутренняя ошибка сервера"
// @Security BearerAuth
// @Router /api/v1/commits [get]
func (h *Handler) List(c *fiber.Ctx) error {
	values, err := h.usecase.List(c.UserContext())
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	items, err := shared.MapValues(values, NewResponse)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.JSON(fiber.Map{"items": items, "total": len(items)})
}

// Get возвращает commit по UUID.
// @Summary Получить коммит
// @Description Возвращает коммит и связанные с ним изменения.
// @ID getCommit
// @Tags Коммиты
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства" example(default)
// @Param id path string true "UUID commit" format(uuid)
// @Success 200 {object} Response "Commit"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Недостаточно прав"
// @Failure 404 {object} shared.ErrorResponse "Ресурс не найден"
// @Failure 500 {object} shared.ErrorResponse "Внутренняя ошибка сервера"
// @Security BearerAuth
// @Router /api/v1/commits/{id} [get]
func (h *Handler) Get(c *fiber.Ctx) error {
	value, err := h.usecase.Get(c.UserContext(), c.Params("id"))
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return h.write(c, fiber.StatusOK, *value)
}

// GetDiff возвращает commit с его изменениями.
// @Summary Получить изменения коммита
// @Description Возвращает коммит с полным списком входящих в него изменений документов.
// @ID getCommitDiff
// @Tags Коммиты
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства" example(default)
// @Param id path string true "UUID commit" format(uuid)
// @Success 200 {object} Response "Коммит и изменения"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Недостаточно прав"
// @Failure 404 {object} shared.ErrorResponse "Ресурс не найден"
// @Failure 500 {object} shared.ErrorResponse "Внутренняя ошибка сервера"
// @Security BearerAuth
// @Router /api/v1/commits/{id}/diff [get]
func (h *Handler) GetDiff(c *fiber.Ctx) error {
	return h.Get(c)
}

// PlanRestore возвращает diff восстановления выбранного коммита.
// @Summary Рассчитать восстановление коммита
// @Description Возвращает план восстановления рабочего пространства до выбранного коммита без записи изменений.
// @ID planCommitRestore
// @Tags Коммиты
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства" example(default)
// @Param id path string true "UUID commit" format(uuid)
// @Success 200 {object} RestorePlanResponse "План восстановления"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Недостаточно прав"
// @Failure 404 {object} shared.ErrorResponse "Ресурс не найден"
// @Failure 409 {object} shared.ErrorResponse "Конфликт состояния"
// @Failure 500 {object} shared.ErrorResponse "Внутренняя ошибка сервера"
// @Security BearerAuth
// @Router /api/v1/commits/{id}/restore/plan [post]
func (h *Handler) PlanRestore(c *fiber.Ctx) error {
	value, err := h.usecase.PlanRestore(c.UserContext(), c.Params("id"))
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.JSON(value)
}

// Restore восстанавливает workspace и создаёт новый коммит.
// @Summary Восстановить коммит
// @Description Восстанавливает состояние как новые ревизии и создаёт новый коммит, не переписывая историю.
// @ID restoreCommit
// @Tags Коммиты
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства" example(default)
// @Accept json
// @Param id path string true "UUID commit" format(uuid)
// @Param request body RestoreRequest true "Ожидаемая head sequence"
// @Success 201 {object} Response "Коммит восстановления"
// @Failure 400 {object} shared.ErrorResponse "Некорректный запрос"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Недостаточно прав"
// @Failure 404 {object} shared.ErrorResponse "Ресурс не найден"
// @Failure 409 {object} shared.ErrorResponse "Конфликт состояния"
// @Failure 500 {object} shared.ErrorResponse "Внутренняя ошибка сервера"
// @Security BearerAuth
// @Router /api/v1/commits/{id}/restore [post]
func (h *Handler) Restore(c *fiber.Ctx) error {
	request, err := shared.DecodeAndValidate[RestoreRequest](c, h.validator)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	value, err := h.usecase.Restore(c.UserContext(), c.Params("id"), *request.ExpectedHeadSequence)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return h.write(c, fiber.StatusCreated, *value)
}

func (h *Handler) write(c *fiber.Ctx, status int, value entities.Commit) error {
	response, err := NewResponse(value)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.Status(status).JSON(response)
}
