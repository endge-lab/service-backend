package postgres

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	apperrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/observability"
	"github.com/endge-lab/service-backend/internal/repo/postgres/mappers"
	"github.com/endge-lab/service-backend/internal/repo/postgres/sqlc"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/endge-lab/service-kit-go/pkg/telemetry"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

var _ ports.DomainDependenciesRepository = (*DomainDependenciesRepository)(nil)

// DomainDependenciesRepository сохраняет derived dependency projection в PostgreSQL.
type DomainDependenciesRepository struct{ *baseRepository }

// NewDomainDependenciesRepository создаёт PostgreSQL-реализацию
// ports.DomainDependenciesRepository.
func NewDomainDependenciesRepository(queries *sqlc.Queries, core *observability.Core, metrics *RepositoryMetrics) *DomainDependenciesRepository {
	return &DomainDependenciesRepository{baseRepository: newBaseRepository(queries, core, metrics, "domain_dependencies")}
}

// ReplaceForOwner полностью заменяет dependency projection одного canonical document.
//
// Параметры:
//
//	ctx               - контекст с WorkspaceScope и optional активной транзакцией;
//	owner             - технический и публичный идентификаторы owner document;
//	references        - уже нормализованные dependencies из extractor;
//	state             - результат проверки canonical source;
//	verificationError - optional безопасное описание неполной проверки.
//
// Что делает функция:
//
//	Проверяет workspace scope, upsert-ит state owner, удаляет прежние dependency
//	строки и сохраняет текущий список references. Самостоятельно transaction не
//	открывает: вызов использует transaction из ctx, чтобы caller мог сохранить
//	canonical document и его projection атомарно.
//
// Возвращаемые значения:
//
//	error - workspace_required, workspace_scope_mismatch, validation_error или storage error.
func (r *DomainDependenciesRepository) ReplaceForOwner(ctx context.Context, owner entities.DomainDependencyOwner, references []entities.DomainDependencyReference, state entities.DomainDependencyVerificationState, verificationError *string) (err error) {
	ctx, step := telemetry.StartTrace(ctx, r.observer.Tracer(), r.observer.Logger(), "repo.domain_dependencies.replace_for_owner")
	defer func() { step.End(err) }()

	workspaceID, err := workspaceIDFromContext(ctx)
	if err != nil {
		return err
	}
	if owner.ID == uuid.Nil {
		return apperrors.InvalidInput("validation_error", "dependency owner id is required")
	}

	queries := r.queries(ctx)
	if err = queries.UpsertDomainDependencyState(ctx, sqlc.UpsertDomainDependencyStateParams{
		WorkspaceID:       workspaceID,
		OwnerType:         owner.Type,
		OwnerID:           owner.ID,
		OwnerIdentity:     owner.Identity,
		VerificationState: string(state),
		VerificationError: mappers.NullableTextToSQLC(verificationError),
	}); err != nil {
		return r.storageError("upsert dependency state", err)
	}
	if err = queries.DeleteDomainDependenciesForOwner(ctx, sqlc.DeleteDomainDependenciesForOwnerParams{WorkspaceID: workspaceID, OwnerType: owner.Type, OwnerID: owner.ID}); err != nil {
		return r.storageError("delete previous dependencies", err)
	}
	for _, reference := range references {
		if err = queries.CreateDomainDependency(ctx, sqlc.CreateDomainDependencyParams{
			WorkspaceID:        workspaceID,
			OwnerType:          owner.Type,
			OwnerID:            owner.ID,
			DependencyType:     reference.Type,
			DependencyIdentity: reference.Identity,
			SourcePath:         reference.SourcePath,
		}); err != nil {
			return r.storageError("create dependency", err)
		}
	}
	return nil
}

// DeleteForOwner удаляет state canonical document и все принадлежащие ему
// dependency строки через database cascade.
func (r *DomainDependenciesRepository) DeleteForOwner(ctx context.Context, ownerType string, ownerID uuid.UUID) (err error) {
	ctx, step := telemetry.StartTrace(ctx, r.observer.Tracer(), r.observer.Logger(), "repo.domain_dependencies.delete_for_owner")
	defer func() { step.End(err) }()

	workspaceID, err := workspaceIDFromContext(ctx)
	if err != nil {
		return err
	}
	if ownerID == uuid.Nil {
		return apperrors.InvalidInput("validation_error", "dependency owner id is required")
	}
	if err = r.queries(ctx).DeleteDomainDependencyStateForOwner(ctx, sqlc.DeleteDomainDependencyStateForOwnerParams{WorkspaceID: workspaceID, OwnerType: ownerType, OwnerID: ownerID}); err != nil {
		return r.storageError("delete dependency state", err)
	}
	return nil
}

// ListUsages читает ограниченную страницу owners, использующих dependency identity.
func (r *DomainDependenciesRepository) ListUsages(ctx context.Context, dependencyType, dependencyIdentity string, options ports.DomainDependenciesListOptions) (result entities.DomainDependencyUsages, err error) {
	ctx, step := telemetry.StartTrace(ctx, r.observer.Tracer(), r.observer.Logger(), "repo.domain_dependencies.list_usages")
	defer func() { step.End(err) }()

	workspaceID, err := workspaceIDFromContext(ctx)
	if err != nil {
		return result, err
	}
	if options.Limit < 0 || options.Offset < 0 {
		return result, apperrors.InvalidInput("validation_error", "usage page values must not be negative")
	}
	values, err := r.queries(ctx).ListDomainDependencyUsages(ctx, sqlc.ListDomainDependencyUsagesParams{
		WorkspaceID: workspaceID, DependencyType: dependencyType, DependencyIdentity: dependencyIdentity,
		LimitCount: int32(options.Limit), OffsetCount: int32(options.Offset),
	})
	if err != nil {
		return result, r.storageError("list dependency usages", err)
	}
	result = entities.DomainDependencyUsages{Items: make([]entities.DomainDependencyUsage, 0, len(values)), Limit: options.Limit, Offset: options.Offset}
	for _, value := range values {
		result.Items = append(result.Items, mappers.DomainDependencyUsage(value))
		result.Total = value.Total
	}
	return result, nil
}

// EnsureNotReferenced получает первые usages для delete guard. Conflict error
// формирует usecase, поскольку только он знает type удаляемой domain entity.
func (r *DomainDependenciesRepository) EnsureNotReferenced(ctx context.Context, dependencyType, dependencyIdentity string, limit int) (entities.DomainDependencyUsages, error) {
	return r.ListUsages(ctx, dependencyType, dependencyIdentity, ports.DomainDependenciesListOptions{Limit: limit})
}

func (r *DomainDependenciesRepository) storageError(operation string, err error) error {
	r.observer.Logger().Error("domain dependency repository operation failed", zap.String("operation", operation), zap.Error(err))
	return apperrors.Internal("internal_error", "failed to persist domain dependencies")
}
