package middleware

import (
	"context"
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/access_control"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/gofiber/fiber/v2"
)

type CurrentUserMiddleware struct{ access *access_control.UseCase }

func NewCurrentUserMiddleware(access *access_control.UseCase) *CurrentUserMiddleware {
	return &CurrentUserMiddleware{access: access}
}

func (m *CurrentUserMiddleware) Resolve() fiber.Handler {
	return func(c *fiber.Ctx) error {
		identity, ok := IdentityFromContext(c.UserContext())
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"code": "unauthorized", "message": "authentication required"})
		}
		user, platformAdmin, err := m.access.ResolveCurrentActor(c.UserContext(), ports.UpsertCurrentUserInput{ProviderID: identity.ProviderID, Subject: identity.Subject, Issuer: identity.Issuer, Username: identity.Username, DisplayName: identity.DisplayName}, identity.PlatformAdmin)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"code": "current_user_failed", "message": "failed to prepare current user"})
		}
		if !user.Active {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"code": "user_inactive", "message": "user is inactive"})
		}
		identity.AuthUserID = strings.TrimSpace(user.ID)
		ctx := context.WithValue(c.UserContext(), currentUserKey, user)
		ctx = context.WithValue(ctx, identityKey, identity)
		ctx = context.WithValue(ctx, userIDKey, user.ID)
		ctx = entities.WithCurrentActor(ctx, entities.CurrentActor{User: user, PlatformAdmin: platformAdmin})
		c.SetUserContext(ctx)
		return c.Next()
	}
}
