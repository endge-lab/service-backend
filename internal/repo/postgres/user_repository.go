package postgres

import (
	"context"
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/repo/postgres/sqlc"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/endge-lab/service-kit-go/pkg/logging"
	"github.com/endge-lab/service-kit-go/pkg/telemetry"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type UserRepository struct {
	*baseRepository
}

func NewUserRepository(queries *sqlc.Queries, tracer trace.Tracer, logger *zap.Logger) *UserRepository {
	return &UserRepository{
		baseRepository: newBaseRepository(queries, tracer, logger, "user"),
	}
}

func (r *UserRepository) SyncUserFromIdentity(ctx context.Context, input ports.SyncUserInput) (user *entities.User, err error) {
	ctx, step := telemetry.StartTrace(
		ctx,
		r.tracer,
		r.logger,
		"repo.user.sync_from_identity",
		attribute.String("repository", "user"),
		attribute.String("auth.user_id", strings.TrimSpace(input.AuthUserID)),
	)
	defer func() {
		step.End(err)
	}()

	logger := logging.WithContext(ctx, r.logger)
	authUserID := strings.TrimSpace(input.AuthUserID)
	if authUserID == "" {
		return nil, domainerrors.ErrAuthUserIDRequired
	}

	logger.Debug("syncing service user from identity", zap.String("auth_user_id", authUserID))

	record, err := r.queries(ctx).UpsertServiceUserFromIdentity(ctx, sqlc.UpsertServiceUserFromIdentityParams{
		ID:          uuid.New(),
		AuthUserID:  authUserID,
		Username:    strings.TrimSpace(input.Username),
		DisplayName: strings.TrimSpace(input.DisplayName),
		Role:        strings.TrimSpace(input.Role),
	})
	if err != nil {
		return nil, mapPostgresError(err, "user.sync_from_identity")
	}

	user = mapServiceUser(record)
	logger.Debug("service user synced", zap.String("service_user_id", user.ID))
	return user, nil
}
