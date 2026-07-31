package postgres

import (
	"errors"

	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	postgresUniqueViolation     = "23505"
	postgresForeignKeyViolation = "23503"
	postgresCheckViolation      = "23514"
	errorDetailConstraint       = "constraint"
)

type postgresErrorKind uint8

const (
	postgresErrorUnknown postgresErrorKind = iota
	postgresErrorUniqueViolation
	postgresErrorForeignKeyViolation
	postgresErrorCheckViolation
)

type postgresErrorInfo struct {
	kind           postgresErrorKind
	constraintName string
}

type storageErrorMapping struct {
	identityConstraintNames []string
	identityConflictMessage string
	validationMessage       string
	internalCode            domainerrors.Code
	internalStorageMessage  string
}

var (
	projectStorageErrorMapping = storageErrorMapping{
		identityConstraintNames: []string{"projects_workspace_identity_unique", "projects_identity_unique"},
		identityConflictMessage: "project identity already exists",
		validationMessage:       "project data violates a constraint",
		internalCode:            "internal_error",
		internalStorageMessage:  "failed to create project",
	}
	workspaceStorageErrorMapping = storageErrorMapping{
		identityConstraintNames: []string{"workspaces_identity_key"},
		identityConflictMessage: "workspace identity already exists",
		validationMessage:       "workspace data violates a constraint",
		internalCode:            "internal_error",
		internalStorageMessage:  "failed to save workspace",
	}
	componentLegacyStorageErrorMapping = storageErrorMapping{
		identityConstraintNames: []string{"components_legacy_project_identity_unique"},
		identityConflictMessage: "component identity already exists",
		validationMessage:       "component data violates a constraint",
		internalCode:            "internal_error",
		internalStorageMessage:  "failed to save component",
	}
	converterStorageErrorMapping = storageErrorMapping{
		identityConstraintNames: []string{"converters_project_identity_unique"},
		identityConflictMessage: "converter identity already exists",
		validationMessage:       "converter data violates a constraint",
		internalCode:            "internal_error",
		internalStorageMessage:  "failed to save converter",
	}
	queryStorageErrorMapping = storageErrorMapping{
		identityConstraintNames: []string{"queries_project_identity_unique"},
		identityConflictMessage: "query identity already exists",
		validationMessage:       "query data violates a constraint",
		internalCode:            "internal_error",
		internalStorageMessage:  "failed to save query",
	}
	dataViewStorageErrorMapping = storageErrorMapping{
		identityConstraintNames: []string{"data_views_project_identity_unique"},
		identityConflictMessage: "data view identity already exists",
		validationMessage:       "data view data violates a constraint",
		internalCode:            "internal_error",
		internalStorageMessage:  "failed to save data view",
	}
	tenantStorageErrorMapping = storageErrorMapping{
		identityConstraintNames: []string{
			"tenants_workspace_identity_unique",
			"tenants_workspace_code_unique",
		},
		identityConflictMessage: "tenant identity or code already exists",
		validationMessage:       "tenant data violates a constraint",
		internalCode:            "internal_error",
		internalStorageMessage:  "failed to save tenant",
	}
)

func classifyPostgresError(err error) postgresErrorInfo {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return postgresErrorInfo{kind: postgresErrorUnknown}
	}

	info := postgresErrorInfo{constraintName: pgErr.ConstraintName}
	switch pgErr.Code {
	case postgresUniqueViolation:
		info.kind = postgresErrorUniqueViolation
	case postgresForeignKeyViolation:
		info.kind = postgresErrorForeignKeyViolation
	case postgresCheckViolation:
		info.kind = postgresErrorCheckViolation
	default:
		info.kind = postgresErrorUnknown
	}

	return info
}

func mapStorageError(err error, mapping storageErrorMapping) error {
	postgresError := classifyPostgresError(err)

	if postgresError.kind == postgresErrorUniqueViolation &&
		containsConstraint(mapping.identityConstraintNames, postgresError.constraintName) {
		return domainerrors.Conflict("identity_conflict", mapping.identityConflictMessage)
	}

	if postgresError.kind == postgresErrorForeignKeyViolation || postgresError.kind == postgresErrorCheckViolation {
		return domainerrors.InvalidInput("validation_error", mapping.validationMessage)
	}

	return domainerrors.Internal(mapping.internalCode, mapping.internalStorageMessage)
}

func containsConstraint(constraints []string, value string) bool {
	for _, constraint := range constraints {
		if constraint == value {
			return true
		}
	}

	return false
}

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
