package bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/endge-lab/service-backend/internal/api/http/middleware"
	"github.com/endge-lab/service-backend/internal/config"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	bridgeplatform "github.com/endge-lab/service-backend/internal/platform/bridge"
	bridgeusecase "github.com/endge-lab/service-backend/internal/usecase/bridge"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"golang.org/x/time/rate"
)

type UseCase interface {
	Join(context.Context, string, string, entities.BridgePrincipal, entities.BridgeHello) error
	Handle(context.Context, string, entities.BridgeMessage) error
	Check(context.Context, string) bool
	Leave(string)
}

func BindUseCase(value *bridgeusecase.UseCase) UseCase { return value }

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

func (h *Handler) upgrade(c *fiber.Ctx, role string, principal entities.BridgePrincipal) error {
	if !h.config.Enabled {
		return fiber.ErrNotFound
	}
	allowed := false
	for _, origin := range h.config.AllowedOrigins {
		if c.Get(fiber.HeaderOrigin) == origin {
			allowed = true
			break
		}
	}
	if !allowed {
		return fiber.ErrForbidden
	}
	if !websocket.IsWebSocketUpgrade(c) {
		return fiber.ErrUpgradeRequired
	}
	// No fiber.Ctx or request-owned buffers are retained after upgrade.
	return websocket.New(func(conn *websocket.Conn) {
		id := uuid.NewString()
		registered := false
		limiter := rate.NewLimiter(10, 20)
		_ = h.connections.Serve(id, conn, func(payload []byte) error {
			if !limiter.Allow() {
				return fmt.Errorf("Bridge message rate exceeded")
			}
			var message entities.BridgeMessage
			if err := json.Unmarshal(payload, &message); err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if !registered {
				if message.Type != "hello" {
					return fmt.Errorf("Bridge hello required")
				}
				var hello entities.BridgeHello
				if err := json.Unmarshal(message.Data, &hello); err != nil {
					return err
				}
				if err := h.usecase.Join(ctx, id, role, principal, hello); err != nil {
					return err
				}
				registered = true
				return nil
			}
			if err := h.usecase.Handle(ctx, id, message); err != nil {
				if message.ID != "" {
					h.connections.Send(id, entities.BridgeMessage{Type: "result", ID: message.ID, Error: err.Error()})
					return nil
				}
				return err
			}
			return nil
		}, func(ctx context.Context) bool { return h.usecase.Check(ctx, id) }, func() { h.usecase.Leave(id) })
	}, websocket.Config{HandshakeTimeout: 10 * time.Second, Origins: h.config.AllowedOrigins})(c)
}
