package usecase

import (
	"context"
	"strings"

	"github.com/endge-lab/service-template-go/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-template-go/internal/domain/errors"
	"github.com/endge-lab/service-template-go/internal/ports"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type LoadSessionInput struct {
	AuthUserID  string
	Username    string
	DisplayName string
	Role        string
	SessionID   string
	App         string
	Platform    string
	Scope       []string
	ExpiresAt   string
}

type LoadSessionOutput struct {
	Session *entities.SessionInfo
	User    *entities.User
}

type LoadSessionUseCase interface {
	Execute(ctx context.Context, input LoadSessionInput) (*LoadSessionOutput, error)
}

type loadSessionUseCase struct {
	observedUseCase
	userRepository ports.UserRepository
}

func NewLoadSessionUseCase(
	userRepository ports.UserRepository,
	tracer trace.Tracer,
	logger *zap.Logger,
	metrics *UseCaseMetrics,
) LoadSessionUseCase {
	return &loadSessionUseCase{
		observedUseCase: newObservedUseCase(
			tracer,
			logger.With(zap.String("component", "usecase"), zap.String("usecase", "load_session")),
			metrics,
		),
		userRepository: userRepository,
	}
}

func (u *loadSessionUseCase) Execute(ctx context.Context, input LoadSessionInput) (output *LoadSessionOutput, err error) {
	ctx, obs := u.startObservedOperation(ctx, "load_session", []attribute.KeyValue{
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
