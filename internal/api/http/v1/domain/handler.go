package domain

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/endge-lab/service-backend/internal/api/http/respond"
	"github.com/endge-lab/service-backend/internal/api/http/v1/shared"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	appvalidator "github.com/endge-lab/service-kit-go/pkg/validator"
	"github.com/gofiber/fiber/v2"
)

// Live возвращает единый рабочий snapshot для инициализации Configurator.
// @Summary Получить актуальное состояние workspace
// @Description Возвращает текущий workspace и все его документы одним JSON с локальными state-полями.
// @ID getLiveDomain
// @Tags Домен
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства" example(default)
// @Success 200 {object} ExportResponse "Рабочий snapshot"
// @Header 200 {string} ETag "Generation и head sequence workspace"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Недостаточно прав"
// @Failure 500 {object} shared.ErrorResponse "Внутренняя ошибка сервера"
// @Security BearerAuth
// @Router /api/v1/domain [get]
func (h *Handler) Live(c *fiber.Ctx) error {
	raw, err := h.usecase.Live(c.UserContext())
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	var snapshot struct {
		Workspace struct {
			State struct {
				Generation   string `json:"generation"`
				HeadSequence int64  `json:"headSequence"`
			} `json:"state"`
		} `json:"workspace"`
	}
	if json.Unmarshal(raw, &snapshot) == nil && snapshot.Workspace.State.Generation != "" {
		c.Set(fiber.HeaderETag, fmt.Sprintf(`"%s:%d"`, snapshot.Workspace.State.Generation, snapshot.Workspace.State.HeadSequence))
	}
	c.Type("json")
	return c.Send(raw)
}

type Handler struct {
	usecase   UseCase
	validator appvalidator.Validator
}

// NewHandler создаёт portable domain HTTP-обработчик.
func NewHandler(usecase UseCase, validator appvalidator.Validator) *Handler {
	return &Handler{usecase: usecase, validator: validator}
}

// Status возвращает версию последнего зафиксированного состояния домена.
// @Summary Получить состояние версии домена
// @Description Возвращает domainVersion только когда текущее workspace не содержит незакоммиченных revisions.
// @ID getDomainStatus
// @Tags Перенос домена
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства" example(default)
// @Success 200 {object} entities.DomainStatus "Состояние версии домена"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Недостаточно прав"
// @Security BearerAuth
// @Router /api/v1/domain/status [get]
func (h *Handler) Status(c *fiber.Ctx) error {
	value, err := h.usecase.Status(c.UserContext())
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.JSON(value)
}

// Export выгружает переносимый JSON текущего workspace.
// @Summary Экспортировать рабочее пространство
// @Description Возвращает переносимый пакет без UUID-связей, пользователей, назначения ролей, истории и секретов.
// @ID exportDomain
// @Tags Перенос домена
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства" example(default)
// @Param download query bool false "Скачать JSON как файл" default(false)
// @Success 200 {object} ExportResponse "Переносимый пакет"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Недостаточно прав"
// @Failure 500 {object} shared.ErrorResponse "Внутренняя ошибка сервера"
// @Security BearerAuth
// @Router /api/v1/domain/export [get]
func (h *Handler) Export(c *fiber.Ctx) error {
	raw, err := h.usecase.Export(c.UserContext())
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	c.Type("json")
	download, err := shared.OptionalBoolQuery(c, "download", false)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	if download {
		identity := "workspace"
		if scope, ok := entities.WorkspaceAccessFromContext(c.UserContext()); ok {
			identity = scope.Workspace.Identity
		}
		c.Attachment(shared.SafeAttachmentName(identity, "workspace") + "-snapshot.json")
	}
	return c.Send(raw)
}

// PlanImport валидирует bundle и возвращает план импорта без изменения данных.
// @Summary Проверить импорт домена
// @Description Валидирует переносимый пакет и возвращает план импорта без изменения данных.
// @ID planDomainImport
// @Tags Перенос домена
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства" example(default)
// @Accept json
// @Param request body entities.PortableBundle true "Полный workspace snapshot"
// @Success 200 {object} ImportPlanResponse "План импорта"
// @Failure 400 {object} shared.ErrorResponse "Некорректный запрос"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Недостаточно прав"
// @Failure 409 {object} shared.ErrorResponse "Конфликт состояния"
// @Failure 500 {object} shared.ErrorResponse "Внутренняя ошибка сервера"
// @Security BearerAuth
// @Router /api/v1/domain/import/plan [post]
func (h *Handler) PlanImport(c *fiber.Ctx) error {
	snapshot, err := shared.DecodeAndValidate[entities.PortableBundle](c, h.validator)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	value, err := h.usecase.PlanImport(c.UserContext(), snapshot)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.JSON(value)
}

// Import атомарно импортирует переносимый пакет и создаёт commit.
// @Summary Импортировать домен
// @Description Атомарно применяет snapshot как новые revisions и создаёт обратимый import commit.
// @ID importDomain
// @Tags Перенос домена
// @Produce json
// @Param X-Endge-Workspace header string true "Identity рабочего пространства" example(default)
// @Accept json
// @Param If-Match header string true "ETag workspace из import plan" example("generation:42")
// @Param request body ImportRequest true "Подтверждение импорта"
// @Success 201 {object} ImportResponse "Результат безопасного импорта"
// @Failure 400 {object} shared.ErrorResponse "Некорректный запрос"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Недостаточно прав"
// @Failure 409 {object} shared.ErrorResponse "Конфликт состояния"
// @Failure 500 {object} shared.ErrorResponse "Внутренняя ошибка сервера"
// @Security BearerAuth
// @Router /api/v1/domain/import [post]
func (h *Handler) Import(c *fiber.Ctx) error {
	request, err := shared.DecodeAndValidate[ImportRequest](c, h.validator)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	ifMatch := strings.TrimSpace(c.Get(fiber.HeaderIfMatch))
	if ifMatch == "" {
		return respond.WriteErrorResponse(c, domainerrors.New("precondition_required", "If-Match header is required", 428))
	}
	value, err := h.usecase.Import(c.UserContext(), request.PlanID, request.Confirmation, ifMatch)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.Status(fiber.StatusCreated).JSON(newImportResponse(*value))
}
