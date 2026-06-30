package postgres

import (
	"context"
	"strings"

	"github.com/endge-lab/service-template-go/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-template-go/internal/domain/errors"
	"github.com/endge-lab/service-template-go/internal/ports"
	"github.com/endge-lab/service-template-go/internal/util"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type UserRepository struct {
	pool   *pgxpool.Pool
	tracer trace.Tracer
	logger *zap.Logger
}

func NewUserRepository(pool *pgxpool.Pool, tracer trace.Tracer, logger *zap.Logger) *UserRepository {
	return &UserRepository{
		pool:   pool,
		tracer: tracer,
		logger: logger.With(zap.String("component", "repo"), zap.String("repository", "user")),
	}
}

func (r *UserRepository) SyncUserFromIdentity(ctx context.Context, input ports.SyncUserInput) (user *entities.User, err error) {
	ctx, step := util.StartTrace(
		ctx,
		r.tracer,
		r.logger,
		"repo.user.sync_from_identity",
		attribute.String("repository", "user"),
		attribute.String("auth.user_id", strings.TrimSpace(input.AuthUserID)),
	)
	defer func() {
		step.EndTrace(err)
	}()

	logger := util.LoggerWithTrace(ctx, r.logger)
	authUserID := strings.TrimSpace(input.AuthUserID)
	if authUserID == "" {
		return nil, domainerrors.ErrAuthUserIDRequired
	}

	logger.Debug("syncing service user from identity", zap.String("auth_user_id", authUserID))

	row := queryRowerFromContext(ctx, r.pool).QueryRow(ctx, `
		INSERT INTO service_users (
			id, auth_user_id, username, display_name, role, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		ON CONFLICT (auth_user_id) DO UPDATE
		SET
			username = CASE
				WHEN EXCLUDED.username <> '' THEN EXCLUDED.username
				ELSE service_users.username
			END,
			display_name = CASE
				WHEN EXCLUDED.display_name <> '' THEN EXCLUDED.display_name
				ELSE service_users.display_name
			END,
			role = CASE
				WHEN EXCLUDED.role <> '' THEN EXCLUDED.role
				ELSE service_users.role
			END,
			updated_at = NOW()
		RETURNING id, auth_user_id, username, display_name, role, created_at, updated_at
	`,
		uuid.New(),
		authUserID,
		strings.TrimSpace(input.Username),
		strings.TrimSpace(input.DisplayName),
		strings.TrimSpace(input.Role),
	)

	user = &entities.User{}
	if err = row.Scan(
		&user.ID,
		&user.AuthUserID,
		&user.Username,
		&user.DisplayName,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		return nil, err
	}

	logger.Debug("service user synced", zap.String("service_user_id", user.ID))
	return user, nil
}
