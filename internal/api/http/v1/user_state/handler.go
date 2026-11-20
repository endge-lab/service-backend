package user_state

import (
	"encoding/json"

	"github.com/endge-lab/service-backend/internal/api/http/middleware"
	"github.com/endge-lab/service-backend/internal/api/http/respond"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	state "github.com/endge-lab/service-backend/internal/usecase/user_state"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	usecase *state.UseCase
}

func NewHandler(usecase *state.UseCase) *Handler { return &Handler{usecase: usecase} }

func RegisterRoutes(router fiber.Router, h *Handler) {
	root := router.Group("/user-state")
	root.Get("/", h.self)
	root.Patch("/", h.saveSelf)
	root.Get("/admin/contexts", h.contexts)
	root.Get("/admin/users", h.users)
	root.Get("/admin/users/:user", h.readUser)
	root.Patch("/admin/users/:user", h.saveUser)
	root.Post("/admin/reset", h.reset)
}

type contextsResponse struct {
	Items   []state.Scope      `json:"items"`
	Current *state.Scope       `json:"current"`
	Catalog []state.Definition `json:"catalog"`
}

type resetRequest struct {
	Partition    string `json:"partition"`
	User         string `json:"userId,omitempty"`
	Confirmation string `json:"confirmation"`
}

type resetResponse struct {
	Deleted int64 `json:"deleted"`
}

func groups(c *fiber.Ctx) []string {
	identity, _ := middleware.IdentityFromContext(c.UserContext())
	return identity.Groups
}

func selections(c *fiber.Ctx) (map[string]string, error) {
	raw := c.Query("facets", "{}")
	var result map[string]string
	if len(raw) > 4096 || json.Unmarshal([]byte(raw), &result) != nil || result == nil {
		return nil, domainerrors.InvalidInput("user_state_context_invalid", "Expected a facet selection object")
	}
	return result, nil
}

func decode(c *fiber.Ctx, target any) error {
	if len(c.Body()) > 64*1024 || json.Unmarshal(c.Body(), target) != nil {
		return domainerrors.InvalidInput("user_state_body_invalid", "Expected a settings request of at most 64 KiB")
	}
	return nil
}

// @Summary Прочитать свои рабочие настройки
// @Tags User State
// @ID getUserState
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

func send(c *fiber.Ctx, data any, err error) error {
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	c.Set(fiber.HeaderCacheControl, "no-store")
	return c.JSON(data)
}
