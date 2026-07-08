package usecase

import (
	"context"
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/adapters"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/endge-lab/service-backend/internal/usecase/shared"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type LoadSessionInput = adapters.LoadSessionInput
type LoadSessionOutput = adapters.LoadSessionOutput
type LoadSessionUseCase = adapters.LoadSessionService

type loadSessionUseCase struct {
	observed       shared.ObservedUseCase
	userRepository ports.UserRepository
}

type LoadSessionParams struct {
	UserRepository ports.UserRepository
	Tracer         trace.Tracer
	Logger         *zap.Logger
	Metrics        *shared.UseCaseMetrics
}

func NewLoadSessionUseCase(
	userRepository ports.UserRepository,
	tracer trace.Tracer,
	logger *zap.Logger,
	metrics *shared.UseCaseMetrics,
) LoadSessionUseCase {
	return newLoadSessionUseCase(LoadSessionParams{
		UserRepository: userRepository,
		Tracer:         tracer,
		Logger:         logger,
		Metrics:        metrics,
	})
}

func newLoadSessionUseCase(params LoadSessionParams) LoadSessionUseCase {
	return &loadSessionUseCase{
		observed: shared.NewObservedUseCase(
			params.Tracer,
			params.Logger.With(zap.String("component", "usecase"), zap.String("usecase", "load_session")),
			params.Metrics,
		),
		userRepository: params.UserRepository,
	}
}

func (u *loadSessionUseCase) Execute(ctx context.Context, input LoadSessionInput) (output *LoadSessionOutput, err error) {
	ctx, obs := u.observed.StartObservedOperation(ctx, "load_session", []attribute.KeyValue{
		attribute.String("auth.user_id", strings.TrimSpace(input.AuthUserID)),
	}, nil)
	defer obs.End(&err)

	logger := obs.Logger()
	logger.Debug("load session use case started", zap.String("auth_user_id", strings.TrimSpace(input.AuthUserID)))

	authUserID := strings.TrimSpace(input.AuthUserID)
	if authUserID == "" {
		return nil, domainerrors.ErrAuthUserIDRequired
	}

	user, err := u.userRepository.SyncUserFromIdentity(ctx, ports.SyncUserInput{
		AuthUserID:  authUserID,
		Username:    input.Username,
		DisplayName: input.DisplayName,
		Role:        input.Role,
	})
	if err != nil {
		return nil, err
	}

	logger.Debug("load session use case completed", zap.String("service_user_id", user.ID))

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
