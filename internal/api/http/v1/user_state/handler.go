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
// @Description Возвращает настройки выбранного пользователя администратору рабочего пространства в доступном ему контексте.
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
// @Description Применяет подтверждённые администратором изменения по ревизиям ключей выбранного пользователя в доступном контексте.
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
// @Description Возвращает доступные администратору контексты, активный контекст и каталог поддерживаемых ключей с их описаниями.
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
	items, err := h.usecase.Scopes(c.UserContext(), groups(c))
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	_, _, current, err := h.usecase.Context(c.UserContext(), facets, groups(c), true)
	var active *state.Scope
	if err == nil {
		active = &current
	} else if code := domainerrors.CodeOf(err); code != "user_state_context_invalid" && code != "user_state_facet_forbidden" {
		return respond.RespondDomainError(c, nil, err)
	}
	return send(c, contextsResponse{Items: items, Current: active, Catalog: state.Catalog}, nil)
}

// @Summary Получить пользователей с настройками
// @Tags User State
// @ID listUserStateUsers
// @Description Показывает пользователей с серверными настройками в выбранном контексте с поиском и постраничной выдачей.
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
	result, err := h.usecase.Users(c.UserContext(), facets, groups(c), c.Query("q"), c.Query("after"))
	return send(c, result, err)
}

func (h *Handler) read(admin bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		facets, err := selections(c)
		if err != nil {
			return respond.RespondDomainError(c, nil, err)
		}
		c.Set(fiber.HeaderCacheControl, "no-store")
		result, err := h.usecase.Read(c.UserContext(), facets, groups(c), c.Params("user"), admin)
		return send(c, result, err)
	}
}

func (h *Handler) save(admin bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		facets, err := selections(c)
		if err != nil {
			return respond.RespondDomainError(c, nil, err)
		}
		var batch state.Batch
		if err := decode(c, &batch); err != nil {
			return respond.RespondDomainError(c, nil, err)
		}
		result, err := h.usecase.Save(c.UserContext(), facets, groups(c), c.Params("user"), admin, batch)
		return send(c, result, err)
	}
}

// @Summary Сбросить пользовательские настройки в контексте
// @Tags User State
// @ID resetUserState
// @Description Сбрасывает настройки выбранного пользователя или всех пользователей в подтверждённом контексте и отзывает прежнюю очередь записей.
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
	count, err := h.usecase.Reset(c.UserContext(), facets, groups(c), input.User, input.Confirmation, input.Partition)
	return send(c, resetResponse{Deleted: count}, err)
}
