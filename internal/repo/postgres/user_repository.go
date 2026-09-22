package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/repo/postgres/sqlc"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

type UserRepository struct{ queries *sqlc.Queries }

func NewUserRepository(queries *sqlc.Queries) *UserRepository {
	return &UserRepository{queries: queries}
}

func (r *UserRepository) SyncUserFromIdentity(ctx context.Context, input ports.SyncUserInput) (*entities.User, error) {
	authID := strings.TrimSpace(input.AuthUserID)
	if authID == "" {
		return nil, fmt.Errorf("auth user id is required")
	}
	user, err := r.UpsertCurrentUser(ctx, ports.UpsertCurrentUserInput{ProviderID: "legacy", Subject: authID, Issuer: "urn:endge:legacy", Username: input.Username, DisplayName: input.DisplayName})
	if err != nil {
		return nil, err
	}
	user.Role = strings.TrimSpace(input.Role)
	return user, nil
}
func (r *UserRepository) UpsertCurrentUser(ctx context.Context, input ports.UpsertCurrentUserInput) (*entities.User, error) {
	queries := r.queries
	if tx, ok := txFromContext(ctx); ok {
		queries = queries.WithTx(tx)
	}
	row, err := queries.UpsertCurrentUser(ctx, sqlc.UpsertCurrentUserParams{ProviderID: strings.TrimSpace(input.ProviderID), Subject: strings.TrimSpace(input.Subject), Issuer: strings.TrimSpace(input.Issuer), Username: strings.TrimSpace(input.Username), DisplayName: strings.TrimSpace(input.DisplayName)})
	if err != nil {
		return nil, err
	}
	return &entities.User{ID: row.ID.String(), ProviderID: row.ProviderID, Subject: row.Subject, Issuer: row.Issuer, AuthUserID: row.Subject, Username: row.Username, DisplayName: row.DisplayName, Active: row.Active, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, LastSeenAt: row.LastSeenAt}, nil
}
