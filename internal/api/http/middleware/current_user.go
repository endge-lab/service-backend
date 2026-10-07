package middleware

import (
	"context"
	"strings"

	"github.com/endge-lab/service-backend/internal/api/http/respond"
	"github.com/endge-lab/service-backend/internal/auth"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/access_control"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/gofiber/fiber/v2"
)

type CurrentUserMiddleware struct {
	access   *access_control.UseCase
	sessions *auth.SessionManager
}

func NewCurrentUserMiddleware(access *access_control.UseCase, sessions *auth.SessionManager) *CurrentUserMiddleware {
	return &CurrentUserMiddleware{access: access, sessions: sessions}
}

func (m *CurrentUserMiddleware) Resolve() fiber.Handler {
	return func(c *fiber.Ctx) error {
		identity, ok := IdentityFromContext(c.UserContext())
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"code": "unauthorized", "message": "authentication required"})
		}
		user, platformAdmin, err := m.access.ResolveCurrentActor(c.UserContext(), ports.UpsertCurrentUserInput{ProviderID: identity.ProviderID, Subject: identity.Subject, Issuer: identity.Issuer, Username: identity.Username, DisplayName: identity.DisplayName}, identity.PlatformAdmin, identity.ExternalAccess)
		if err != nil && domainerrors.CodeOf(err) == "external_access_stale" && identity.SessionID != "" && identity.ExternalAccess != nil {
			refreshed, refreshErr := m.sessions.ResolveAfterStale(c.UserContext(), c.Cookies(m.sessions.CookieName()), identity.ExternalAccess.TokenHash)
			if refreshErr != nil {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"code": "unauthorized", "message": "authentication required", "loginUrl": m.sessions.LoginURL(),
				})
			}
			identity = requestIdentity(refreshed.Claims, refreshed.SessionID)
			user, platformAdmin, err = m.access.ResolveCurrentActor(c.UserContext(), ports.UpsertCurrentUserInput{
				ProviderID: identity.ProviderID, Subject: identity.Subject, Issuer: identity.Issuer,
				Username: identity.Username, DisplayName: identity.DisplayName,
			}, identity.PlatformAdmin, identity.ExternalAccess)
		}
		if err != nil {
			return respond.RespondDomainError(c, nil, err)
		}
		if !user.Active {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"code": "user_inactive", "message": "user is inactive"})
		}
		identity.AuthUserID = strings.TrimSpace(user.ID)
		identity.PlatformAdmin = platformAdmin
		ctx := context.WithValue(c.UserContext(), currentUserKey, user)
		ctx = context.WithValue(ctx, identityKey, identity)
		ctx = context.WithValue(ctx, userIDKey, user.ID)
		ctx = entities.WithCurrentActor(ctx, entities.CurrentActor{User: user, PlatformAdmin: platformAdmin})
		c.SetUserContext(ctx)
		return c.Next()
	}
}
