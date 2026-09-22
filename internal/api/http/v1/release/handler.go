package release

import (
	"github.com/endge-lab/service-backend/internal/api/http/respond"
	"github.com/endge-lab/service-backend/internal/api/http/v1/shared"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	appvalidator "github.com/endge-lab/service-kit-go/pkg/validator"
	"github.com/gofiber/fiber/v2"
)

// Handler обслуживает immutable release HTTP-операции.
type Handler struct {
	usecase   UseCase
	validator appvalidator.Validator
}

// NewHandler создаёт release HTTP-обработчик.
func NewHandler(usecase UseCase, validator appvalidator.Validator) *Handler {
	return &Handler{usecase: usecase, validator: validator}
}

// Create создаёт release из существующего workspace commit.
// @Summary Создать релиз
// @Description Создаёт неизменяемый переносимый снимок из существующего коммита.
// @ID createRelease
// @Tags Релизы
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства" example(default)
// @Accept json
// @Param request body CreateRequest true "Данные release"
// @Success 201 {object} Response "Релиз создан"
// @Failure 400 {object} shared.ErrorResponse "Некорректный запрос"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Недостаточно прав"
// @Failure 404 {object} shared.ErrorResponse "Ресурс не найден"
// @Failure 409 {object} shared.ErrorResponse "Конфликт состояния"
// @Failure 500 {object} shared.ErrorResponse "Внутренняя ошибка сервера"
// @Security BearerAuth
// @Router /api/v1/releases [post]
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

// List возвращает releases текущего workspace.
// @Summary Получить релизы
// @Description Возвращает неизменяемые релизы текущего рабочего пространства.
// @ID listReleases
// @Tags Релизы
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства" example(default)
// @Success 200 {object} ListResponse "Список релизов"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Недостаточно прав"
// @Failure 500 {object} shared.ErrorResponse "Внутренняя ошибка сервера"
// @Security BearerAuth
// @Router /api/v1/releases [get]
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

// Get возвращает release по identity.
// @Summary Получить релиз
// @Description Возвращает метаданные релиза по identity.
// @ID getRelease
// @Tags Релизы
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства" example(default)
// @Param identity path string true "Identity release" maxlength(160)
// @Success 200 {object} Response "Релиз"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Недостаточно прав"
// @Failure 404 {object} shared.ErrorResponse "Ресурс не найден"
// @Failure 500 {object} shared.ErrorResponse "Внутренняя ошибка сервера"
// @Security BearerAuth
// @Router /api/v1/releases/{identity} [get]
func (h *Handler) Get(c *fiber.Ctx) error {
	identity, err := shared.PathParam(c, "identity")
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	value, err := h.usecase.Get(c.UserContext(), identity)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return h.write(c, fiber.StatusOK, *value)
}

// Export отдаёт сохранённый переносимый JSON без replay revisions.
// @Summary Экспортировать релиз
// @Description Возвращает сохранённый переносимый JSON релиза без replay истории; identity=last выбирает последний релиз.
// @ID exportRelease
// @Tags Релизы
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства" example(default)
// @Param identity path string true "Identity release или last" maxlength(160)
// @Param If-None-Match header string false "ETag ранее полученного release" example("\"sha256:0123456789abcdef\"")
// @Param download query bool false "Скачать JSON как файл" default(false)
// @Success 200 {object} ExportResponse "Portable snapshot"
// @Header 200 {string} ETag "Checksum релиза"
// @Header 200 {string} Cache-Control "private, no-cache"
// @Header 200 {string} Vary "X-Endge-Workspace, Authorization, Cookie"
// @Success 304 "Release не изменился"
// @Header 304 {string} ETag "Checksum релиза"
// @Header 304 {string} Cache-Control "private, no-cache"
// @Header 304 {string} Vary "X-Endge-Workspace, Authorization, Cookie"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Недостаточно прав"
// @Failure 404 {object} shared.ErrorResponse "Ресурс не найден"
// @Failure 500 {object} shared.ErrorResponse "Внутренняя ошибка сервера"
// @Security BearerAuth
// @Router /api/v1/releases/{identity}/export [get]
func (h *Handler) Export(c *fiber.Ctx) error {
	download, err := shared.OptionalBoolQuery(c, "download", false)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	identity, err := shared.PathParam(c, "identity")
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	metadata, err := h.usecase.Get(c.UserContext(), identity)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	etag := `"` + metadata.Checksum + `"`
	c.Set(fiber.HeaderETag, etag)
	c.Set(fiber.HeaderCacheControl, "private, no-cache")
	c.Set(fiber.HeaderVary, "X-Endge-Workspace, Authorization, Cookie")
	if shared.IfNoneMatch(c, etag) {
		return c.Status(fiber.StatusNotModified).Send(nil)
	}
	artifact, err := h.usecase.GetArtifact(c.UserContext(), *metadata)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	c.Type("json")
	if download {
		c.Attachment(shared.SafeAttachmentName(artifact.Identity, "release") + "-release.json")
	}
	return c.Send(artifact.Data)
}

// PlanRestore возвращает diff восстановления release.
// @Summary Рассчитать восстановление релиза
// @Description Возвращает план восстановления релиза без записи изменений.
// @ID planReleaseRestore
// @Tags Релизы
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства" example(default)
// @Param identity path string true "Identity release" maxlength(160)
// @Success 200 {object} RestorePlanResponse "План восстановления"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Недостаточно прав"
// @Failure 404 {object} shared.ErrorResponse "Ресурс не найден"
// @Failure 409 {object} shared.ErrorResponse "Конфликт состояния"
// @Failure 500 {object} shared.ErrorResponse "Внутренняя ошибка сервера"
// @Security BearerAuth
// @Router /api/v1/releases/{identity}/restore/plan [post]
func (h *Handler) PlanRestore(c *fiber.Ctx) error {
	identity, err := shared.PathParam(c, "identity")
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	value, err := h.usecase.PlanRestore(c.UserContext(), identity)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.JSON(value)
}

// Restore восстанавливает release и создаёт новый workspace commit.
// @Summary Восстановить релиз
// @Description Применяет переносимый снимок как новые ревизии и создаёт коммит восстановления.
// @ID restoreRelease
// @Tags Релизы
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства" example(default)
// @Accept json
// @Param identity path string true "Identity release" maxlength(160)
// @Param request body RestoreRequest true "Ожидаемая head sequence"
// @Success 201 {object} RestoreResponse "Коммит восстановления"
// @Failure 400 {object} shared.ErrorResponse "Некорректный запрос"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Недостаточно прав"
// @Failure 404 {object} shared.ErrorResponse "Ресурс не найден"
// @Failure 409 {object} shared.ErrorResponse "Конфликт состояния"
// @Failure 500 {object} shared.ErrorResponse "Внутренняя ошибка сервера"
// @Security BearerAuth
// @Router /api/v1/releases/{identity}/restore [post]
func (h *Handler) Restore(c *fiber.Ctx) error {
	request, err := shared.DecodeAndValidate[RestoreRequest](c, h.validator)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	identity, err := shared.PathParam(c, "identity")
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	value, err := h.usecase.Restore(c.UserContext(), identity, *request.ExpectedHeadSequence)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.Status(fiber.StatusCreated).JSON(value)
}

func (h *Handler) write(c *fiber.Ctx, status int, value entities.Release) error {
	response, err := NewResponse(value)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.Status(status).JSON(response)
}
