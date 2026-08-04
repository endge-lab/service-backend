package workspace

import (
	"github.com/endge-lab/service-backend/internal/api/http/respond"
	"github.com/endge-lab/service-backend/internal/api/http/v1/shared"
	appvalidator "github.com/endge-lab/service-kit-go/pkg/validator"
	"github.com/gofiber/fiber/v2"
)

// Handler обслуживает workspace и назначение роли HTTP-операции.
type Handler struct {
	usecase   UseCase
	validator appvalidator.Validator
}

// NewHandler создаёт workspace HTTP-обработчик.
func NewHandler(usecase UseCase, validator appvalidator.Validator) *Handler {
	return &Handler{usecase: usecase, validator: validator}
}

// List возвращает доступные текущему пользователю рабочие пространства.
// @Summary Получить доступные рабочие пространства
// @Description Возвращает рабочие пространства, в которых текущий пользователь имеет назначение роли.
// @ID listWorkspaces
// @Tags Рабочие пространства
// @Produce json
// @Success 200 {object} ListResponse "Список рабочих пространств"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 500 {object} shared.ErrorResponse "Внутренняя ошибка сервера"
// @Security BearerAuth
// @Router /api/v1/workspaces [get]
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

// Create проверяет запрос и создаёт рабочее пространство.
// @Summary Создать рабочее пространство
// @Description Создаёт рабочее пространство. Операция доступна администратору платформы.
// @ID createWorkspace
// @Tags Рабочие пространства
// @Accept json
// @Produce json
// @Param request body CreateRequest true "Данные рабочего пространства"
// @Success 201 {object} Response "Рабочее пространство создано"
// @Header 201 {string} ETag "Текущая revision"
// @Failure 400 {object} shared.ErrorResponse "Некорректный запрос"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Требуются права администратору платформы"
// @Failure 409 {object} shared.ErrorResponse "Identity уже занят"
// @Failure 500 {object} shared.ErrorResponse "Внутренняя ошибка сервера"
// @Security BearerAuth
// @Router /api/v1/workspaces [post]
func (h *Handler) Create(c *fiber.Ctx) error {
	request, err := shared.DecodeAndValidate[CreateRequest](c, h.validator)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	value, err := h.usecase.Create(c.UserContext(), request.Input())
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	response, err := NewResponse(*value)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	c.Set(fiber.HeaderETag, shared.ETag(value.Revision))
	return c.Status(fiber.StatusCreated).JSON(response)
}

// Get возвращает рабочее пространство по identity.
// @Summary Получить рабочее пространство
// @Description Возвращает доступное текущему пользователю рабочее пространство по identity.
// @ID getWorkspace
// @Tags Рабочие пространства
// @Produce json
// @Param identity path string true "Identity рабочего пространства" maxlength(160)
// @Success 200 {object} Response "Рабочее пространство"
// @Header 200 {string} ETag "Текущая revision"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Нет доступа"
// @Failure 404 {object} shared.ErrorResponse "Рабочее пространство не найдено"
// @Failure 500 {object} shared.ErrorResponse "Внутренняя ошибка сервера"
// @Security BearerAuth
// @Router /api/v1/workspaces/{identity} [get]
func (h *Handler) Get(c *fiber.Ctx) error {
	value, err := h.usecase.Get(c.UserContext(), c.Params("identity"))
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	response, err := NewResponse(*value)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	c.Set(fiber.HeaderETag, shared.ETag(value.Revision))
	return c.JSON(response)
}

// Patch проверяет If-Match и изменяет рабочее пространство.
// @Summary Изменить рабочее пространство
// @Description Частично изменяет рабочее пространство; revision передаётся в If-Match.
// @ID patchWorkspace
// @Tags Рабочие пространства
// @Accept json
// @Produce json
// @Param identity path string true "Identity рабочего пространства" maxlength(160)
// @Param If-Match header string true "Текущая revision" example("3")
// @Param request body PatchRequest true "Изменяемые поля"
// @Success 200 {object} Response "Рабочее пространство изменено"
// @Header 200 {string} ETag "Новая revision"
// @Failure 400 {object} shared.ErrorResponse "Некорректный запрос"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Нет прав на изменение"
// @Failure 404 {object} shared.ErrorResponse "Рабочее пространство не найдено"
// @Failure 409 {object} shared.ErrorResponse "Конфликт revision или identity"
// @Failure 428 {object} shared.ErrorResponse "Требуется If-Match"
// @Failure 500 {object} shared.ErrorResponse "Внутренняя ошибка сервера"
// @Security BearerAuth
// @Router /api/v1/workspaces/{identity} [patch]
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
	response, err := NewResponse(*value)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	c.Set(fiber.HeaderETag, shared.ETag(value.Revision))
	return c.JSON(response)
}

// ListMembers возвращает явные назначение роли рабочего пространства.
// @Summary Получить назначения ролей рабочего пространства
// @Description Возвращает явно назначенные роли пользователей.
// @ID listWorkspaceMembers
// @Tags Участники рабочего пространства
// @Produce json
// @Param identity path string true "Identity рабочего пространства" maxlength(160)
// @Success 200 {object} MembershipListResponse "Список назначения ролей"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Нет доступа"
// @Failure 404 {object} shared.ErrorResponse "Рабочее пространство не найдено"
// @Failure 500 {object} shared.ErrorResponse "Внутренняя ошибка сервера"
// @Security BearerAuth
// @Router /api/v1/workspaces/{identity}/members [get]
func (h *Handler) ListMembers(c *fiber.Ctx) error {
	values, err := h.usecase.ListMemberships(c.UserContext(), c.Params("identity"))
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	items, err := shared.MapValues(values, NewMembershipResponse)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.JSON(fiber.Map{"items": items, "total": len(items)})
}

// PutMember создаёт или заменяет роль пользователя в workspace.
// @Summary Назначить роль пользователю
// @Description Создаёт назначение роли или полностью заменяет роль указанного пользователя.
// @ID putWorkspaceMember
// @Tags Участники рабочего пространства
// @Accept json
// @Produce json
// @Param identity path string true "Identity рабочего пространства" maxlength(160)
// @Param userId path string true "UUID пользователя" format(uuid)
// @Param request body MembershipRequest true "Назначаемая роль"
// @Success 200 {object} MembershipResponse "Membership обновлён"
// @Failure 400 {object} shared.ErrorResponse "Некорректный запрос"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Нет прав на управление назначения ролей"
// @Failure 404 {object} shared.ErrorResponse "Рабочее пространство или пользователь не найдены"
// @Failure 500 {object} shared.ErrorResponse "Внутренняя ошибка сервера"
// @Security BearerAuth
// @Router /api/v1/workspaces/{identity}/members/{userId} [put]
func (h *Handler) PutMember(c *fiber.Ctx) error {
	request, err := shared.DecodeAndValidate[MembershipRequest](c, h.validator)
	if err != nil {
		return respond.WriteErrorResponse(c, err)
	}
	value, err := h.usecase.PutMembership(c.UserContext(), c.Params("identity"), c.Params("userId"), request.Role)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	response, err := NewMembershipResponse(*value)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.JSON(response)
}

// DeleteMember удаляет явный назначение роли пользователя.
// @Summary Удалить назначение роли пользователя
// @Description Удаляет явно назначеную роль пользователя в рабочем пространстве.
// @ID deleteWorkspaceMember
// @Tags Участники рабочего пространства
// @Produce json
// @Param identity path string true "Identity рабочего пространства" maxlength(160)
// @Param userId path string true "UUID пользователя" format(uuid)
// @Success 204 "Membership удалён"
// @Failure 401 {object} shared.ErrorResponse "Требуется аутентификация"
// @Failure 403 {object} shared.ErrorResponse "Нет прав на управление назначения ролей"
// @Failure 404 {object} shared.ErrorResponse "Membership не найден"
// @Failure 500 {object} shared.ErrorResponse "Внутренняя ошибка сервера"
// @Security BearerAuth
// @Router /api/v1/workspaces/{identity}/members/{userId} [delete]
func (h *Handler) DeleteMember(c *fiber.Ctx) error {
	if err := h.usecase.DeleteMembership(c.UserContext(), c.Params("identity"), c.Params("userId")); err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
