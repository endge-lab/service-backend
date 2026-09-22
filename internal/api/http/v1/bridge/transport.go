package bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"golang.org/x/time/rate"
)

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
