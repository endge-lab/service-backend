package postgres

import (
	"errors"

	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func isIdentityConflict(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == postgresUniqueViolation && pgErr.ConstraintName == constraint
}

const (
	postgresUniqueViolation     = "23505"
	postgresForeignKeyViolation = "23503"
	postgresCheckViolation      = "23514"
	errorDetailConstraint       = "constraint"
)

func mapPostgresError(err error, codePrefix string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domainerrors.ErrNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case postgresUniqueViolation:
			return domainerrors.WithDetails(
				domainerrors.Conflict(domainerrors.Code(codePrefix+".conflict"), "Запись уже существует"),
				map[string]any{errorDetailConstraint: pgErr.ConstraintName},
			)
		case postgresForeignKeyViolation, postgresCheckViolation:
			return domainerrors.WithDetails(
				domainerrors.InvalidInput(domainerrors.Code(codePrefix+".constraint_failed"), "Нарушено ограничение данных"),
				map[string]any{errorDetailConstraint: pgErr.ConstraintName},
			)
		}
	}

	return domainerrors.Internal(domainerrors.Code(codePrefix+".storage_failed"), "Ошибка хранилища данных")
}
