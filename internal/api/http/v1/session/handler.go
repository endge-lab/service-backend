package session

import (
	"github.com/endge-lab/service-backend/internal/api/http/respond"
	"github.com/gofiber/fiber/v2"
)

// Handler обслуживает сведения о текущей пользовательской сессии.
type Handler struct{ usecase UseCase }

// NewHandler создаёт session HTTP-обработчик.
func NewHandler(usecase UseCase) *Handler { return &Handler{usecase: usecase} }

// Current возвращает текущего локального пользователя, platform-admin flag и доступные workspace.
// @Summary Получить текущую сессию
// @Description Возвращает локальную проекцию пользователя, признак администратору платформы и доступные рабочие пространства.
// @ID getCurrentSession
// @Tags Сессия
// @Produce json
// @Success 200 {object} Response "Текущая сессия"
// @Failure 401 {object} respond.ErrorResponse "Требуется аутентификация"
// @Failure 500 {object} respond.ErrorResponse "Внутренняя ошибка сервера"
// @Security BearerAuth
// @Router /api/session/me [get]
func (h *Handler) Current(c *fiber.Ctx) error {
	value, err := h.usecase.Current(c.UserContext())
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	response, err := NewResponse(*value)
	if err != nil {
		return respond.RespondDomainError(c, nil, err)
	}
	return c.JSON(response)
}

// AuthSession — браузерная точка входа Configurator в тот же ответ, что и GET /api/session/me.
// Маршрут находится вне /api: его вызывает Configurator после входа через /auth/login.
// @Summary Получить текущую сессию (браузерный вход)
// @Description Тот же ответ, что у GET /api/session/me, по пути браузерного потока /auth/*.
// @Description Принимает HttpOnly cookie сессии после /auth/login или заголовок Authorization: Bearer.
// @Description При отсутствии аутентификации ответ 401 содержит code=unauthorized и loginUrl для повторного входа.
// @ID getAuthSession
// @Tags Авторизация
// @Produce json
// @Success 200 {object} Response "Текущая сессия"
// @Failure 401 {object} respond.ErrorResponse "Требуется аутентификация (code=unauthorized, loginUrl)"
// @Failure 403 {object} respond.ErrorResponse "Пользователь деактивирован или запрос отклонён политикой доступа"
// @Failure 500 {object} respond.ErrorResponse "Внутренняя ошибка сервера"
// @Security BearerAuth
// @Router /auth/session [get]
func (h *Handler) AuthSession(c *fiber.Ctx) error { return h.Current(c) }
