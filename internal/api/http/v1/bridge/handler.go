package bridge

import (
	"time"

	"github.com/endge-lab/service-backend/internal/api/http/middleware"
	"github.com/endge-lab/service-backend/internal/config"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	bridgeplatform "github.com/endge-lab/service-backend/internal/platform/bridge"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	usecase     UseCase
	connections *bridgeplatform.Connections
	config      config.BridgeConfig
}

func NewHandler(usecase UseCase, connections *bridgeplatform.Connections, cfg *config.Config) *Handler {
	return &Handler{usecase: usecase, connections: connections, config: cfg.Bridge}
}

// Client exposes only dev discovery/consented debug, independent of application Keycloak.
func (h *Handler) Client(c *fiber.Ctx) error {
	if !h.config.DebugEnabled {
		return fiber.ErrNotFound
	}
	return h.upgrade(c, "client", entities.BridgePrincipal{})
}

// Configurator receives identity exclusively from existing authenticated HTTP middleware.
func (h *Handler) Configurator(c *fiber.Ctx) error {
	identity, ok := middleware.IdentityFromContext(c.UserContext())
	if !ok {
		return fiber.ErrUnauthorized
	}
	expires, _ := time.Parse(time.RFC3339, identity.ExpiresAt)
	return h.upgrade(c, "configurator", entities.BridgePrincipal{UserID: identity.AuthUserID, SessionID: identity.SessionID, ExpiresAt: expires})
}
