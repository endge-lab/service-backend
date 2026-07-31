package dependencies

import (
	"context"
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	apperrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/observability"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const deleteGuardUsageLimit = 20

const (
	defaultUsagesLimit = 50
	maxUsagesLimit     = 200
)

// Dependencies координирует derived dependency projection и delete guards.
type Dependencies struct {
	repository ports.DomainDependenciesRepository
	tx         ports.TxManager
	observer   observability.Observer
}

type DependenciesParams struct {
	Repository    ports.DomainDependenciesRepository
	TxManager     ports.TxManager
	Observability *observability.Core
	Metrics       *shared.UseCaseMetrics
}

type ListUsagesInput struct {
	DependencyType     string
	DependencyIdentity string
	Limit              *int
	Offset             *int
}

func NewDependenciesService(params DependenciesParams) *Dependencies {
	return &Dependencies{
		repository: params.Repository,
		tx:         params.TxManager,
		observer:   params.Observability.For(observability.LayerUseCase, "dependencies_usecase").WithRecorder(params.Metrics),
	}
}

// ReplaceForOwner транзакционно заменяет dependency projection одного owner.
//
// Параметры:
//
//	ctx              - контекст с обязательным WorkspaceScope;
//	owner            - owner canonical document;
//	extractionResult - типизированный результат разбора source или authoring JSON.
//
// Что делает функция:
//
//	Валидирует owner и результат extraction, открывает транзакцию через
//	TxManager и заменяет state и dependencies owner. Для сохранения canonical
//	document в той же транзакции entity-usecase должен вызвать
//	ReplaceForOwnerInTransaction внутри собственного transaction callback.
//
// Возвращаемые значения:
//
//	error - validation_error, workspace_required или ошибка repository/транзакции.
func (s *Dependencies) ReplaceForOwner(ctx context.Context, owner entities.DomainDependencyOwner, extractionResult DependencyExtractionResult) (err error) {
	ctx, observed := s.observer.Start(ctx, "dependencies.replace_for_owner", nil, nil)
	defer observed.End(&err)
	owner, references, state, verificationError, err := normalizeReplacement(owner, extractionResult)
	if err != nil {
		return err
	}
	if _, err = workspaceID(ctx); err != nil {
		return err
	}
	if err = s.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		return s.repository.ReplaceForOwner(txCtx, owner, references, state, verificationError)
	}); err != nil {
		return err
	}
	observed.RecordStep("dependencies.replace_for_owner.persisted", "dependency projection replaced", nil, zap.String("owner_type", owner.Type), zap.String("owner_id", owner.ID.String()))
	return nil
}

// ReplaceForOwnerInTransaction заменяет dependencies owner в уже открытой транзакции.
//
// Параметры:
//
//	ctx              - контекст с WorkspaceScope и активной транзакцией;
//	owner            - owner canonical document;
//	extractionResult - типизированный результат разбора source или authoring JSON.
//
// Что делает функция:
//
//	Нормализует и валидирует owner/references, проверяет workspace scope и
//	передаёт projection в repository без открытия новой транзакции. Метод нужен
//	entity-usecase, который сохраняет canonical document и projection атомарно.
//
// Возвращаемые значения:
//
//	error - validation_error, workspace_required или repository error.
func (s *Dependencies) ReplaceForOwnerInTransaction(ctx context.Context, owner entities.DomainDependencyOwner, extractionResult DependencyExtractionResult) error {
	owner, references, state, verificationError, err := normalizeReplacement(owner, extractionResult)
	if err != nil {
		return err
	}
	if _, err = workspaceID(ctx); err != nil {
		return err
	}
	return s.repository.ReplaceForOwner(ctx, owner, references, state, verificationError)
}

// DeleteForOwner транзакционно удаляет dependency state удалённого owner.
//
// Параметры:
//
//	ctx       - контекст с обязательным WorkspaceScope;
//	ownerType - тип удаляемого canonical document;
//	ownerID   - технический UUID удаляемого owner.
//
// Что делает функция:
//
//	Нормализует owner type, проверяет input и workspace scope, затем в
//	транзакции удаляет owner state. Связанные domain_dependencies удаляются
//	PostgreSQL cascade.
//
// Возвращаемые значения:
//
//	error - validation_error, workspace_required или repository/transaction error.
func (s *Dependencies) DeleteForOwner(ctx context.Context, ownerType string, ownerID uuid.UUID) (err error) {
	ctx, observed := s.observer.Start(ctx, "dependencies.delete_for_owner", nil, nil)
	defer observed.End(&err)
	ownerType = strings.TrimSpace(ownerType)
	if ownerType == "" || ownerID == uuid.Nil {
		return apperrors.InvalidInput("validation_error", "dependency owner type and id are required")
	}
	if _, err = workspaceID(ctx); err != nil {
		return err
	}
	return s.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		return s.repository.DeleteForOwner(txCtx, ownerType, ownerID)
	})
}

// ListUsages возвращает страницу документов, ссылающихся на dependency identity.
//
// Параметры:
//
//	ctx   - контекст с обязательным WorkspaceScope;
//	input - dependency type/identity и optional limit/offset страницы.
//
// Что делает функция:
//
//	Нормализует обязательные identity, применяет limit=50 по умолчанию и
//	ограничивает limit значением 200. После проверки workspace scope вызывает
//	repository и возвращает пустой items slice, а не null, если usages нет.
//
// Возвращаемые значения:
//
//	entities.DomainDependencyUsages - items, total, limit и offset;
//	error                           - validation_error, workspace_required или repository error.
func (s *Dependencies) ListUsages(ctx context.Context, input ListUsagesInput) (result entities.DomainDependencyUsages, err error) {
	ctx, observed := s.observer.Start(ctx, "dependencies.list_usages", nil, nil)
	defer observed.End(&err)
	input.DependencyType = strings.TrimSpace(input.DependencyType)
	input.DependencyIdentity = strings.TrimSpace(input.DependencyIdentity)
	if input.DependencyType == "" || input.DependencyIdentity == "" {
		return result, apperrors.InvalidInput("validation_error", "dependency type and dependency identity are required")
	}
	limit, offset, err := normalizeUsagesPage(input.Limit, input.Offset)
	if err != nil {
		return result, err
	}
	if _, err = workspaceID(ctx); err != nil {
		return result, err
	}
	result, err = s.repository.ListUsages(ctx, input.DependencyType, input.DependencyIdentity, ports.DomainDependenciesListOptions{Limit: limit, Offset: offset})
	if err != nil {
		return result, err
	}
	if result.Items == nil {
		result.Items = []entities.DomainDependencyUsage{}
	}
	result.Limit = limit
	result.Offset = offset
	return result, nil
}

// EnsureNotReferenced блокирует hard-delete, если dependency identity ещё
// используется документами текущего workspace.
//
// Параметры:
//
//	ctx                - контекст с обязательным WorkspaceScope;
//	entityType         - тип удаляемой сущности для формирования <entity>_in_use;
//	dependencyType     - тип проверяемой dependency;
//	dependencyIdentity - public identity проверяемой dependency.
//
// Что делает функция:
//
//	Нормализует input, запрашивает первые 20 usages и их полное количество. При
//	наличии usages формирует 409 conflict с code <entity>_in_use и details.usages
//	и details.total; repository не знает type удаляемой entity и этот error не
//	создаёт.
//
// Возвращаемые значения:
//
//	error - validation_error, workspace_required, <entity>_in_use или repository error.
func (s *Dependencies) EnsureNotReferenced(ctx context.Context, entityType, dependencyType, dependencyIdentity string) (err error) {
	ctx, observed := s.observer.Start(ctx, "dependencies.ensure_not_referenced", nil, nil)
	defer observed.End(&err)
	entityType = strings.TrimSpace(entityType)
	dependencyType = strings.TrimSpace(dependencyType)
	dependencyIdentity = strings.TrimSpace(dependencyIdentity)
	if entityType == "" || dependencyType == "" || dependencyIdentity == "" {
		return apperrors.InvalidInput("validation_error", "entity type, dependency type and dependency identity are required")
	}
	if _, err = workspaceID(ctx); err != nil {
		return err
	}
	usages, err := s.repository.EnsureNotReferenced(ctx, dependencyType, dependencyIdentity, deleteGuardUsageLimit)
	if err != nil {
		return err
	}
	if usages.Total == 0 {
		return nil
	}
	return apperrors.WithDetails(
		apperrors.Conflict(apperrors.Code(entityType+"_in_use"), "entity is referenced by other documents"),
		map[string]any{"usages": usages.Items, "total": usages.Total},
	)
}

func normalizeReplacement(owner entities.DomainDependencyOwner, result DependencyExtractionResult) (entities.DomainDependencyOwner, []entities.DomainDependencyReference, entities.DomainDependencyVerificationState, *string, error) {
	owner.Type = strings.TrimSpace(owner.Type)
	owner.Identity = strings.TrimSpace(owner.Identity)
	if owner.Type == "" || owner.Identity == "" || owner.ID == uuid.Nil {
		return entities.DomainDependencyOwner{}, nil, "", nil, apperrors.InvalidInput("validation_error", "dependency owner type, id and identity are required")
	}
	if result.VerificationState != entities.DomainDependencyVerificationStateVerified && result.VerificationState != entities.DomainDependencyVerificationStateUnverified {
		return entities.DomainDependencyOwner{}, nil, "", nil, apperrors.InvalidInput("validation_error", "dependency verification state is invalid")
	}
	if result.VerificationState == entities.DomainDependencyVerificationStateVerified && result.VerificationError != nil {
		return entities.DomainDependencyOwner{}, nil, "", nil, apperrors.InvalidInput("validation_error", "verified dependency extraction cannot contain an error")
	}
	var verificationError *string
	if result.VerificationError != nil {
		value := strings.TrimSpace(*result.VerificationError)
		if value == "" {
			return entities.DomainDependencyOwner{}, nil, "", nil, apperrors.InvalidInput("validation_error", "dependency verification error cannot be empty")
		}
		verificationError = &value
	}
	references := make([]entities.DomainDependencyReference, 0, len(result.References))
	seen := make(map[entities.DomainDependencyReference]struct{}, len(result.References))
	for _, reference := range result.References {
		reference.Type = strings.TrimSpace(reference.Type)
		reference.Identity = strings.TrimSpace(reference.Identity)
		reference.SourcePath = strings.TrimSpace(reference.SourcePath)
		if reference.Type == "" || reference.Identity == "" || reference.SourcePath == "" {
			return entities.DomainDependencyOwner{}, nil, "", nil, apperrors.InvalidInput("validation_error", "dependency type, identity and source path are required")
		}
		if _, exists := seen[reference]; exists {
			continue
		}
		seen[reference] = struct{}{}
		references = append(references, reference)
	}
	return owner, references, result.VerificationState, verificationError, nil
}

func workspaceID(ctx context.Context) (uuid.UUID, error) {
	value, ok := entities.WorkspaceIDFromContext(ctx)
	if !ok {
		return uuid.Nil, apperrors.InvalidInput("workspace_required", "workspace scope is required")
	}
	return value, nil
}

func normalizeUsagesPage(limitInput, offsetInput *int) (int, int, error) {
	limit := defaultUsagesLimit
	if limitInput != nil {
		limit = *limitInput
	}
	if limit <= 0 || limit > maxUsagesLimit {
		return 0, 0, apperrors.InvalidInput("validation_error", "usage limit must be between 1 and 200")
	}
	offset := 0
	if offsetInput != nil {
		offset = *offsetInput
	}
	if offset < 0 {
		return 0, 0, apperrors.InvalidInput("validation_error", "usage offset must not be negative")
	}
	return limit, offset, nil
}
