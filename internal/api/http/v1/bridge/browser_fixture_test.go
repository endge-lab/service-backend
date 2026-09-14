package bridge

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/endge-lab/service-backend/internal/config"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	platform "github.com/endge-lab/service-backend/internal/platform/bridge"
	usecase "github.com/endge-lab/service-backend/internal/usecase/bridge"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/gofiber/fiber/v2"
)

type browserAccess struct{ ports.BridgeAccessRepository }

func (browserAccess) BridgeUser(_ context.Context, id, _ string) (entities.Actor, error) {
	return entities.Actor{ID: id, DisplayName: "Inspection test"}, nil
}

type browserWorkspaces struct{ ports.WorkspaceRepository }

func (browserWorkspaces) GetWorkspace(_ context.Context, id string) (*entities.Workspace, error) {
	if id != "inspection-fixture" {
		return nil, fmt.Errorf("unknown isolated workspace")
	}
	return &entities.Workspace{ID: id, Identity: id, DisplayName: "Inspection test", Active: true}, nil
}
func (browserWorkspaces) WorkspaceRole(context.Context, string, string, bool) (string, error) {
	return "editor", nil
}

type browserGrants struct{ ports.AccessControlRepository }

func (browserGrants) IsPlatformAdmin(context.Context, string) (bool, error) { return false, nil }

// Test-only loopback server: real HTTP/WebSocket/consent/usecase, isolated in-memory identity repositories.
// No production route or authentication bypass is compiled into the application.
func TestBrowserInspectionServer(t *testing.T) {
	port := os.Getenv("ENDGE_INSPECTION_E2E_PORT")
	if port == "" {
		t.Skip("started explicitly by the Playwright webServer fixture")
	}
	if port != "4174" {
		t.Fatal("fixture port must be 4174")
	}
	connections := platform.NewConnections()
	core := usecase.NewUseCase(connections, browserAccess{}, browserWorkspaces{}, browserGrants{}, true)
	cfg := &config.Config{}
	cfg.Bridge.Enabled = true
	cfg.Bridge.DebugEnabled = true
	cfg.Bridge.AllowedOrigins = []string{"http://127.0.0.1:4173"}
	handler := NewHandler(core, connections, cfg)
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/health", func(c *fiber.Ctx) error { return c.SendString("ready") })
	app.Get("/api/v1/bridge/client", handler.Client)
	app.Get("/api/v1/bridge/configurator", func(c *fiber.Ctx) error {
		return handler.upgrade(c, "configurator", entities.BridgePrincipal{UserID: "inspection-user", SessionID: "inspection-auth", ExpiresAt: time.Now().Add(time.Hour)})
	})
	done := make(chan error, 1)
	go func() { done <- app.Listen("127.0.0.1:" + port) }()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	defer func() {
		_ = app.ShutdownWithTimeout(time.Second)
		shutdown, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = connections.Shutdown(shutdown)
	}()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
			return
		case <-ticker.C:
			core.Sweep()
		}
	}
}
