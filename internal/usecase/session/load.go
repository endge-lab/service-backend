package session

import (
	"context"
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/observability"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/endge-lab/service-backend/internal/usecase/shared"

	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

type LoadSession struct {
	observer       observability.Observer
	userRepository ports.UserRepository
}

type LoadSessionParams struct {
	UserRepository ports.UserRepository
	Observability  *observability.Core
	Metrics        *shared.UseCaseMetrics
}

func NewLoadSessionUseCase(
	userRepository ports.UserRepository,
	core *observability.Core,
	metrics *shared.UseCaseMetrics,
) *LoadSession {
	return newLoadSessionUseCase(LoadSessionParams{
		UserRepository: userRepository,
		Observability:  core,
		Metrics:        metrics,
	})
}

func newLoadSessionUseCase(params LoadSessionParams) *LoadSession {
	return &LoadSession{
		observer:       params.Observability.For(observability.LayerUseCase, "load_session_usecase").WithRecorder(params.Metrics),
		userRepository: params.UserRepository,
	}
}

func (u *LoadSession) Execute(ctx context.Context, input LoadSessionInput) (output *LoadSessionOutput, err error) {
	ctx, obs := u.observer.Start(ctx, "load_session", []attribute.KeyValue{
		attribute.String("auth.user_id", strings.TrimSpace(input.AuthUserID)),
	}, nil)
	defer obs.End(&err)

	authUserID := strings.TrimSpace(input.AuthUserID)
	if authUserID == "" {
		return nil, domainerrors.ErrAuthUserIDRequired
	}
	obs.RecordStep("session.load.auth_user_validated", "authenticated user identifier validated", nil,
		zap.String("auth_user_id", authUserID),
	)

	user, err := u.userRepository.SyncUserFromIdentity(ctx, ports.SyncUserInput{
		AuthUserID:  authUserID,
		Username:    input.Username,
		DisplayName: input.DisplayName,
		Role:        input.Role,
	})
	if err != nil {
		return nil, err
	}

	obs.RecordStep("session.load.user_synchronized", "session user synchronized", nil,
		zap.String("service_user_id", user.ID),
	)

	return &LoadSessionOutput{
		Session: &entities.SessionInfo{
			ID:        strings.TrimSpace(input.SessionID),
			SessionID: strings.TrimSpace(input.SessionID),
			App:       strings.TrimSpace(input.App),
			Platform:  strings.TrimSpace(input.Platform),
			Scope:     input.Scope,
			ExpiresAt: strings.TrimSpace(input.ExpiresAt),
		},
		User: user,
	}, nil
}
