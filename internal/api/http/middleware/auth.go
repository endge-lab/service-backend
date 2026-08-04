package middleware

import (
	"context"
	"fmt"
	"strings"

	"github.com/endge-lab/service-backend/internal/auth"
	"github.com/endge-lab/service-backend/internal/config"
	"github.com/gofiber/fiber/v2"
)

type AuthMiddleware interface{ AuthMiddleware() fiber.Handler }

type authMiddleware struct {
	resolver auth.Resolver
	sessions *auth.SessionManager
	config   *config.Config
}

func NewAuthMiddleware(resolver auth.Resolver, sessions *auth.SessionManager, cfg *config.Config) AuthMiddleware {
	return &authMiddleware{resolver: resolver, sessions: sessions, config: cfg}
}

func (m *authMiddleware) AuthMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, sessionID, cookieAuth, err := m.authenticate(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"code": "unauthorized", "message": "authentication required", "loginUrl": m.sessions.LoginURL(),
			})
		}
		if cookieAuth && !isSafeMethod(c.Method()) {
			origin := strings.TrimSpace(c.Get(fiber.HeaderOrigin))
			if origin == "" || !isOriginAllowed(origin, m.config.HTTP.CORSAllowedOrigins) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"code": "csrf_origin_rejected", "message": "request origin is not allowed"})
			}
		}
		identity := RequestIdentity{ProviderID: claims.ProviderID, Subject: claims.Subject, Issuer: claims.Issuer, AuthUserID: claims.Subject,
			Username: claims.Username, DisplayName: claims.DisplayName, Groups: claims.Groups, PlatformAdmin: claims.PlatformAdmin,
			SessionID: sessionID, ExpiresAt: claims.ExpiresAt.Format("2006-01-02T15:04:05Z07:00")}
		ctx := context.WithValue(c.UserContext(), identityKey, identity)
		if sessionID != "" {
			ctx = context.WithValue(ctx, sessionIDKey, sessionID)
		}
		c.SetUserContext(ctx)
		return c.Next()
	}
}

func (m *authMiddleware) authenticate(c *fiber.Ctx) (auth.Claims, string, bool, error) {
	header := strings.TrimSpace(c.Get(fiber.HeaderAuthorization))
	if header != "" {
		if !strings.HasPrefix(strings.ToLower(header), "bearer ") {
			return auth.Claims{}, "", false, fmt.Errorf("unsupported Authorization scheme")
		}
		claims, err := m.resolver.Resolve(c.UserContext(), strings.TrimSpace(header[7:]))
		return claims, "", false, err
	}
	if cookieToken := strings.TrimSpace(c.Cookies(m.sessions.CookieName())); cookieToken != "" {
		identity, err := m.sessions.Resolve(c.UserContext(), cookieToken)
		return identity.Claims, identity.SessionID, true, err
	}
	claims, err := m.resolver.Resolve(c.UserContext(), "")
	return claims, "", false, err
}

func isSafeMethod(method string) bool {
	switch method {
	case fiber.MethodGet, fiber.MethodHead, fiber.MethodOptions:
		return true
	default:
		return false
	}
}
