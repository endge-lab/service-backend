package bootstrap

import (
	"context"
	"time"

	"github.com/endge-lab/service-backend/internal/config"
	platformbridge "github.com/endge-lab/service-backend/internal/platform/bridge"
	"github.com/endge-lab/service-backend/internal/usecase/bridge"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"go.uber.org/fx"
)

func newBridgeUseCase(connections *platformbridge.Connections, access ports.BridgeAccessRepository, workspaces ports.WorkspaceRepository, grants ports.AccessControlRepository, cfg *config.Config) *bridge.UseCase {
	return bridge.NewUseCase(connections, access, workspaces, grants, cfg.Bridge.DebugEnabled && !cfg.App.IsProduction())
}

func registerBridgeLifecycle(lifecycle fx.Lifecycle, usecase *bridge.UseCase, connections *platformbridge.Connections, cfg *config.Config) {
	var cancel context.CancelFunc
	var done chan struct{}
	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			if !cfg.Bridge.Enabled {
				return nil
			}
			var ctx context.Context
			ctx, cancel = context.WithCancel(context.Background())
			done = make(chan struct{})
			go func() {
				defer close(done)
				ticker := time.NewTicker(5 * time.Second)
				defer ticker.Stop()
				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						usecase.Sweep()
					}
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if cancel != nil {
				cancel()
				<-done
			}
			return connections.Shutdown(ctx)
		},
	})
}
