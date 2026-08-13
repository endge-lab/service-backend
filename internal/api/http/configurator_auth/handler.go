package configurator_auth

import (
	"time"

	"github.com/endge-lab/service-backend/internal/auth"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

const loginTransactionCookie = "endge_configurator_login"

type Handler struct {
	sessions *auth.SessionManager
	logger   *zap.Logger
}

func NewHandler(sessions *auth.SessionManager, logger *zap.Logger) *Handler {
	return &Handler{sessions: sessions, logger: logger}
}

// Login начинает настроенный внешний процесс авторизации.
// @Summary Начать вход в Configurator
// @Description Перенаправляет браузер на внешний OIDC-провайдер или dev-адаптер.
// @ID configuratorLogin
// @Tags Авторизация
// @Param returnTo query string false "Безопасный URL возврата после входа"
// @Success 302 "Перенаправление на провайдера"
// @Failure 503 {object} map[string]string "Провайдер временно недоступен"
// @Router /auth/login [get]
func (h *Handler) Login(c *fiber.Ctx) error {
	start, err := h.sessions.Begin(c.UserContext(), c.Query("returnTo"))
	if err != nil {
		h.logger.Warn("failed to start Configurator login", zap.Error(err))
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"code": "auth_login_unavailable", "message": "login is temporarily unavailable"})
	}
	if start.BrowserNonce != "" {
		h.setLoginTransactionCookie(c, start.BrowserNonce, start.ExpiresAt)
	}
	return c.Redirect(start.Location, fiber.StatusFound)
}

// Callback завершает вход и создаёт непрозрачную серверную сессию.
// @Summary Завершить вход в Configurator
// @Description Обменивает authorization code, сохраняет серверную сессию и устанавливает HttpOnly cookie.
// @ID configuratorAuthCallback
// @Tags Авторизация
// @Param state query string true "OIDC state"
// @Param code query string true "Authorization code"
// @Param error query string false "Ошибка внешнего провайдера"
// @Success 303 "Перенаправление в Configurator"
// @Failure 401 {object} map[string]string "Вход отклонён"
// @Router /auth/callback [get]
func (h *Handler) Callback(c *fiber.Ctx) error {
	if providerError := c.Query("error"); providerError != "" {
		h.clearLoginTransactionCookie(c)
		h.logger.Warn("Configurator login rejected by provider", zap.String("provider_error", providerError))
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"code": "auth_login_rejected", "message": "login was rejected"})
	}
	cookieToken, returnURL, expiresAt, err := h.sessions.Complete(
		c.UserContext(), c.Query("state"), c.Query("code"), c.Cookies(loginTransactionCookie),
	)
	h.clearLoginTransactionCookie(c)
	if err != nil {
		h.logger.Warn("failed to complete Configurator login", zap.Error(err))
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"code": "auth_login_failed", "message": "login could not be completed"})
	}
	h.setSessionCookie(c, cookieToken, expiresAt)
	return c.Redirect(returnURL, fiber.StatusSeeOther)
}

func (h *Handler) setLoginTransactionCookie(c *fiber.Ctx, value string, expiresAt time.Time) {
	c.Cookie(&fiber.Cookie{
		Name: loginTransactionCookie, Value: value, Path: h.sessions.LoginCallbackPath(), Domain: h.sessions.CookieDomain(),
		Expires: expiresAt, HTTPOnly: true, Secure: h.sessions.CookieSecure(), SameSite: h.sessions.CookieSameSite(),
	})
}

func (h *Handler) clearLoginTransactionCookie(c *fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name: loginTransactionCookie, Value: "", Path: h.sessions.LoginCallbackPath(), Domain: h.sessions.CookieDomain(),
		Expires: time.Unix(0, 0), MaxAge: -1, HTTPOnly: true, Secure: h.sessions.CookieSecure(), SameSite: h.sessions.CookieSameSite(),
	})
}

// Logout отзывает локальную сессию и удаляет browser cookie.
// @Summary Выйти из Configurator
// @Description Удаляет серверную сессию и при поддержке уведомляет внешний провайдер.
// @ID configuratorLogout
// @Tags Авторизация
// @Success 204 "Сессия завершена"
// @Router /auth/logout [post]
func (h *Handler) Logout(c *fiber.Ctx) error {
	cookieToken := c.Cookies(h.sessions.CookieName())
	if err := h.sessions.Revoke(c.UserContext(), cookieToken); err != nil {
		h.logger.Warn("provider logout failed after local session revocation", zap.Error(err))
	}
	h.clearSessionCookie(c)
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) setSessionCookie(c *fiber.Ctx, value string, expiresAt time.Time) {
	c.Cookie(&fiber.Cookie{
		Name:     h.sessions.CookieName(),
		Value:    value,
		Path:     "/",
		Domain:   h.sessions.CookieDomain(),
		Expires:  expiresAt,
		HTTPOnly: true,
		Secure:   h.sessions.CookieSecure(),
		SameSite: h.sessions.CookieSameSite(),
	})
}

func (h *Handler) clearSessionCookie(c *fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name:     h.sessions.CookieName(),
		Value:    "",
		Path:     "/",
		Domain:   h.sessions.CookieDomain(),
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HTTPOnly: true,
		Secure:   h.sessions.CookieSecure(),
		SameSite: h.sessions.CookieSameSite(),
	})
}
