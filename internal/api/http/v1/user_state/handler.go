package user_state

import (
	"github.com/endge-lab/service-backend/internal/api/http/respond"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	state "github.com/endge-lab/service-backend/internal/usecase/user_state"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	usecase UseCase
}

func NewHandler(usecase UseCase) *Handler { return &Handler{usecase: usecase} }

// @Summary Прочитать свои рабочие настройки
// @Tags User State
// @ID getUserState
// @Description Возвращает настройки текущей учётной записи и ревизию сброса в выбранном контексте рабочего пространства.
// @Produce json
// @Param X-Endge-Workspace header string true "Workspace identity"
// @Param facets query string false "JSON object of selected facet documents"
// @Param application query string false "Application identity for personal settings shared across workspaces"
// @Success 200 {object} state.Snapshot
// @Failure 403 {object} respond.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/user-state [get]
func (h *Handler) self(c *fiber.Ctx) error { return h.read(false)(c) }

// @Summary Сохранить свои рабочие настройки
// @Tags User State
// @ID saveUserState
// @Description Применяет пакет операций текущей учётной записи с проверкой контекста, ревизии сброса и ревизий ключей.
// @Produce json
// @Param X-Endge-Workspace header string true "Workspace identity"
// @Param facets query string false "JSON object of selected facet documents"
// @Param application query string false "Application identity for personal settings shared across workspaces"
// @Accept json
// @Param payload body state.Batch true "Settings request"
// @Success 200 {object} state.BatchResult
// @Failure 403 {object} respond.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/user-state [patch]
func (h *Handler) saveSelf(c *fiber.Ctx) error { return h.save(false)(c) }

// @Summary Прочитать настройки пользователя
// @Tags User State
// @ID adminReadUserState
// @Description Возвращает настройки выбранного пользователя участнику рабочего пространства в выбранном контексте.
// @Produce json
// @Param X-Endge-Workspace header string true "Workspace identity"
// @Param facets query string false "JSON object of selected facet documents"
// @Param user path string true "Service user UUID"
// @Success 200 {object} state.Snapshot
// @Failure 403 {object} respond.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/user-state/admin/users/{user} [get]
func (h *Handler) readUser(c *fiber.Ctx) error { return h.read(true)(c) }

// @Summary Сохранить настройки пользователя
// @Tags User State
// @ID adminSaveUserState
// @Description Применяет изменения по ревизиям ключей выбранного пользователя. Требуется роль Editor или Admin в рабочем пространстве.
// @Produce json
// @Param X-Endge-Workspace header string true "Workspace identity"
// @Param facets query string false "JSON object of selected facet documents"
// @Param user path string true "Service user UUID"
// @Accept json
// @Param payload body state.Batch true "Settings request"
// @Success 200 {object} state.BatchResult
// @Failure 403 {object} respond.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/user-state/admin/users/{user} [patch]
func (h *Handler) saveUser(c *fiber.Ctx) error { return h.save(true)(c) }

// @Summary Получить разрешённые контексты и каталог настроек
// @Tags User State
// @ID listUserStateContexts
// @Description Возвращает участнику рабочего пространства контексты с сохранёнными настройками, активный контекст и каталог поддерживаемых ключей.
// @Produce json
// @Param X-Endge-Workspace header string true "Workspace identity"
// @Param facets query string false "JSON object of selected facet documents"
// @Success 200 {object} contextsResponse
// @Failure 403 {object} respond.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/user-state/admin/contexts [get]
func (h *Handler) contexts(c *fiber.Ctx) error {
	facets, err := selections(c)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	items, err := h.usecase.Scopes(c.UserContext())
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	_, _, current, err := h.usecase.Context(c.UserContext(), facets)
	var active *state.Scope
	if err == nil {
		active = &current
	} else if domainerrors.CodeOf(err) != "user_state_context_invalid" {
		return respond.RespondDomainError(c, nil, err)
	}
	return send(c, contextsResponse{Items: items, Current: active, Catalog: state.Catalog}, nil)
}

// @Summary Получить пользователей с настройками
// @Tags User State
// @ID listUserStateUsers
// @Description Показывает участнику рабочего пространства пользователей с серверными настройками в выбранном контексте с поиском и постраничной выдачей.
// @Produce json
// @Param X-Endge-Workspace header string true "Workspace identity"
// @Param facets query string false "JSON object of selected facet documents"
// @Param q query string false "Username or display name"
// @Param after query string false "User cursor"
// @Success 200 {object} state.UserPage
// @Failure 403 {object} respond.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/user-state/admin/users [get]
func (h *Handler) users(c *fiber.Ctx) error {
	facets, err := selections(c)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	result, err := h.usecase.Users(c.UserContext(), facets, c.Query("q"), c.Query("after"))
	return send(c, result, err)
}

func (h *Handler) read(managed bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		facets, err := selections(c)
		if err != nil {
			return respond.RespondDomainError(c, nil, err)
		}
		c.Set(fiber.HeaderCacheControl, "no-store")
		result, err := h.usecase.Read(c.UserContext(), facets, c.Params("user"), managed, c.Query("application"))
		return send(c, result, err)
	}
}

func (h *Handler) save(managed bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		facets, err := selections(c)
		if err != nil {
			return respond.RespondDomainError(c, nil, err)
		}
		var batch state.Batch
		if err := decode(c, &batch); err != nil {
			return respond.RespondDomainError(c, nil, err)
		}
		result, err := h.usecase.Save(c.UserContext(), facets, c.Params("user"), managed, c.Query("application"), batch)
		return send(c, result, err)
	}
}

// @Summary Сбросить пользовательские настройки в контексте
// @Tags User State
// @ID resetUserState
// @Description Сбрасывает настройки выбранного пользователя или всех пользователей в подтверждённом контексте. Требуется роль Admin в рабочем пространстве.
// @Produce json
// @Param X-Endge-Workspace header string true "Workspace identity"
// @Param facets query string false "JSON object of selected facet documents"
// @Accept json
// @Param payload body resetRequest true "Settings request"
// @Success 200 {object} resetResponse
// @Failure 403 {object} respond.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/user-state/admin/reset [post]
func (h *Handler) reset(c *fiber.Ctx) error {
	facets, err := selections(c)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	var input resetRequest
	if err := decode(c, &input); err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	count, err := h.usecase.Reset(c.UserContext(), facets, input.User, input.Confirmation, input.Partition)
	return send(c, resetResponse{Deleted: count}, err)
}
