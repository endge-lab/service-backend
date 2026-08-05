package bootstrap

import (
	"context"
	"fmt"
	"time"

	"github.com/endge-lab/service-backend/internal/auth"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// registerAuthSessionCleanup привязывает очистку просроченного временного
// auth-состояния к lifecycle приложения, а не к пользовательскому login request.
func registerAuthSessionCleanup(lifecycle fx.Lifecycle, sessions *auth.SessionManager, logger *zap.Logger) {
	var cancel context.CancelFunc
	var done chan struct{}
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			interval := sessions.CleanupInterval()
			if interval <= 0 {
				return fmt.Errorf("Configurator auth cleanup interval must be positive")
			}
			if err := sessions.Cleanup(ctx); err != nil {
				return err
			}
			workerContext, workerCancel := context.WithCancel(context.Background())
			cancel = workerCancel
			done = make(chan struct{})
			go runAuthSessionCleanup(workerContext, done, interval, sessions, logger)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if cancel == nil {
				return nil
			}
			cancel()
			select {
			case <-done:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	})
}

func runAuthSessionCleanup(ctx context.Context, done chan<- struct{}, interval time.Duration, sessions *auth.SessionManager, logger *zap.Logger) {
	defer close(done)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := sessions.Cleanup(ctx); err != nil && ctx.Err() == nil {
				logger.Warn("failed to clean expired Configurator auth state", zap.Error(err))
			}
		}
	}
}
