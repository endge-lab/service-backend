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
