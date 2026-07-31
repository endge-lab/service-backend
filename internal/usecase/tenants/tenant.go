package tenants

import (
	"context"
	"strings"
	"time"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	apperrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/observability"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	relationresolver "github.com/endge-lab/service-backend/internal/usecase/relations"
	"github.com/endge-lab/service-backend/internal/usecase/shared"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

const operationTimeout = 15 * time.Second

type Tenant struct {
	tenants   ports.TenantsRepository
	folders   ports.FoldersRepository
	tx        ports.TxManager
	relations *relationresolver.Resolver
	observer  observability.Observer
}

type TenantParams struct {
	Repository       ports.TenantsRepository
	FolderRepository ports.FoldersRepository
	TxManager        ports.TxManager
	Relations        *relationresolver.Resolver
	Observability    *observability.Core
	Metrics          *shared.UseCaseMetrics
}

func NewTenantService(params TenantParams) *Tenant {
	resolver := params.Relations
	if resolver == nil {
		resolver = relationresolver.NewResolver(nil, params.FolderRepository)
	}
	return &Tenant{
		tenants:   params.Repository,
		folders:   params.FolderRepository,
		tx:        params.TxManager,
		relations: resolver,
		observer:  params.Observability.For(observability.LayerUseCase, "tenants_usecase").WithRecorder(params.Metrics),
	}
}

// Create валидирует вход tenant и сохраняет его в текущем workspace.
//
// Параметры:
//
//	ctx - контекст с обязательным WorkspaceScope;
//	input - public identity, displayName, code, optional description,
//	optional folderIdentity и обязательный configuration contribution.
//
// Что делает функция:
//
//	Нормализует обязательные строки и структурно валидирует contribution.
//	В одной транзакции получает или создаёт system folder root-tenants,
//	разрешает folderIdentity внутри того же workspace и передаёт в repository
//	уже собранный RTenant с техническим FolderID. При отсутствии folderIdentity
//	использует root-tenants. UUID workspace и папки из input не принимаются.
//
// Возвращаемые значения:
//
//	*TenantView - созданный tenant с ID, FolderID, timestamps и public folderIdentity;
//	error - validation, folder, conflict или persistence error.
func (s *Tenant) Create(ctx context.Context, input CreateTenantInput) (result *TenantView, err error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	ctx, observed := s.observer.Start(ctx, "tenant.create", nil, nil)
	defer observed.End(&err)

	if err = normalizeCreateInput(&input); err != nil {
		return nil, err
	}
	workspaceID, err := workspaceID(ctx)
	if err != nil {
		return nil, err
	}
	if err = s.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		root, rootErr := s.ensureRootFolder(txCtx, workspaceID)
		if rootErr != nil {
			return rootErr
		}
		folderID, folderErr := s.resolveFolderID(txCtx, workspaceID, input.FolderIdentity, root)
		if folderErr != nil {
			return folderErr
		}
		created, createErr := s.tenants.Create(txCtx, &entities.RTenant{
			WorkspaceID: workspaceID, Identity: input.Identity, DisplayName: input.DisplayName, Code: input.Code,
			Description: input.Description, FolderID: folderID, Configuration: *input.Configuration,
		})
		if createErr != nil {
			return createErr
		}
		result, err = s.toView(txCtx, created)
		return err
	}); err != nil {
		observed.Logger().Error("tenant create failed", zap.Error(err))
		return nil, err
	}
	observed.RecordStep("tenant.create.persisted", "tenant created", nil, zap.String("tenant_id", result.Tenant.ID.String()), zap.String("tenant_identity", result.Tenant.Identity))
	return result, nil
}

// List возвращает tenants текущего workspace с опциональной фильтрацией по папке.
//
// Параметры:
//
//	ctx - контекст с обязательным WorkspaceScope;
//	input - optional folderIdentity, не содержащий технический UUID.
//
// Что делает функция:
//
//	Проверяет workspace scope. Если передан folderIdentity, разрешает его
//	только среди workspace-level folders типа tenants и передаёт полученный UUID
//	в repository filter. Без filter возвращает все tenants текущего workspace.
//
// Возвращаемые значения:
//
//	[]*TenantView - tenants с public folderIdentity и без effective configuration;
//	error - workspace_required, validation, folder not_found или repository error.
func (s *Tenant) List(ctx context.Context, input ListTenantsInput) (result []*TenantView, err error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	ctx, observed := s.observer.Start(ctx, "tenant.list", nil, nil)
	defer observed.End(&err)
	workspaceID, err := workspaceID(ctx)
	if err != nil {
		return nil, err
	}
	var folderID *uuid.UUID
	if input.FolderIdentity != nil {
		identity := strings.TrimSpace(*input.FolderIdentity)
		if identity == "" {
			return nil, apperrors.InvalidInput("validation_error", "tenant folder identity is required")
		}
		folder, folderErr := s.resolveTenantFolder(ctx, workspaceID, identity)
		if folderErr != nil {
			return nil, folderErr
		}
		folderID = &folder.ID
	}
	items, err := s.tenants.List(ctx, ports.TenantsFilter{FolderID: folderID})
	if err != nil {
		return nil, err
	}
	result = make([]*TenantView, 0, len(items))
	for _, item := range items {
		view, viewErr := s.toView(ctx, item)
		if viewErr != nil {
			return nil, viewErr
		}
		result = append(result, view)
	}
	observed.RecordStep("tenant.list.result_loaded", "tenants listed", nil, zap.Int("count", len(result)))
	return result, nil
}

// GetByIdentity возвращает tenant по стабильному identity внутри текущего workspace.
//
// Параметры:
//
//	ctx - контекст с обязательным WorkspaceScope;
//	identity - public stable identity tenant.
//
// Что делает функция:
//
//	Удаляет внешние пробелы, отклоняет пустой identity и выполняет scoped lookup
//	через repository. Tenant из другого workspace не раскрывается: repository
//	вернёт тот же not_found, что и для отсутствующей записи.
//
// Возвращаемые значения:
//
//	*TenantView - найденная запись с public folderIdentity;
//	error - validation_error, not_found или ошибка repository.
func (s *Tenant) GetByIdentity(ctx context.Context, identity string) (result *TenantView, err error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	identity = strings.TrimSpace(identity)
	ctx, observed := s.observer.Start(ctx, "tenant.get_by_identity", nil, nil)
	defer observed.End(&err)
	if _, err = workspaceID(ctx); err != nil {
		return nil, err
	}
	if identity == "" {
		return nil, apperrors.InvalidInput("validation_error", "tenant identity is required")
	}
	tenant, err := s.tenants.GetByIdentity(ctx, identity)
	if err != nil {
		return nil, err
	}
	result, err = s.toView(ctx, tenant)
	if err != nil {
		return nil, err
	}
	observed.RecordStep("tenant.get_by_identity.result_loaded", "tenant retrieved", nil, zap.String("tenant_id", result.Tenant.ID.String()), zap.String("tenant_identity", result.Tenant.Identity))
	return result, nil
}

// Update применяет partial patch к tenant из текущего workspace.
//
// Параметры:
//
//	ctx - контекст с обязательным WorkspaceScope;
//	input - identity tenant и поля PATCH, где NullableString различает
//	отсутствующее поле и явный null.
//
// Что делает функция:
//
//	Нормализует переданные поля и запрещает пустой patch. В транзакции получает
//	текущий tenant, сохраняет его ID, WorkspaceID, Identity и CreatedAt, затем
//	заменяет только поля из input. Description:null очищает описание;
//	folderIdentity:null разрешается в root-tenants; configuration при передаче
//	целиком заменяет previous contribution без JSON merge. Полная effective
//	configuration здесь не вычисляется и не сохраняется.
//
// Возвращаемые значения:
//
//	*TenantView - обновлённая запись с public folderIdentity;
//	error - validation, not_found, folder, conflict или persistence error.
func (s *Tenant) Update(ctx context.Context, input UpdateTenantInput) (result *TenantView, err error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	ctx, observed := s.observer.Start(ctx, "tenant.update", nil, nil)
	defer observed.End(&err)
	if err = normalizeUpdateInput(&input); err != nil {
		return nil, err
	}
	workspaceID, err := workspaceID(ctx)
	if err != nil {
		return nil, err
	}
	if err = s.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		current, getErr := s.tenants.GetByIdentity(txCtx, input.Identity)
		if getErr != nil {
			return getErr
		}
		updated := *current
		if input.DisplayName != nil {
			updated.DisplayName = *input.DisplayName
		}
		if input.Code != nil {
			updated.Code = *input.Code
		}
		if input.Description.Set {
			updated.Description = input.Description.Value
		}
		if input.ConfigurationSet {
			updated.Configuration = *input.Configuration
		}
		if input.FolderIdentity.Set {
			root, rootErr := s.ensureRootFolder(txCtx, workspaceID)
			if rootErr != nil {
				return rootErr
			}
			folderID, folderErr := s.resolveFolderID(txCtx, workspaceID, input.FolderIdentity.Value, root)
			if folderErr != nil {
				return folderErr
			}
			updated.FolderID = folderID
		}
		persisted, updateErr := s.tenants.Update(txCtx, &updated)
		if updateErr != nil {
			return updateErr
		}
		result, err = s.toView(txCtx, persisted)
		return err
	}); err != nil {
		return nil, err
	}
	observed.RecordStep("tenant.update.persisted", "tenant updated", nil, zap.String("tenant_id", result.Tenant.ID.String()), zap.String("tenant_identity", result.Tenant.Identity))
	return result, nil
}

// HardDelete физически удаляет tenant по identity из текущего workspace.
//
// Параметры:
//
//	ctx - контекст с обязательным WorkspaceScope;
//	identity - public stable identity удаляемого tenant.
//
// Что делает функция:
//
//	Нормализует identity, проверяет scope и вызывает scoped delete repository.
//	Tenant из другого workspace намеренно не раскрывается и выглядит как
//	обычная not_found ошибка.
//
// Возвращаемые значения:
//
//	error - validation_error, not_found или persistence error.
func (s *Tenant) HardDelete(ctx context.Context, identity string) (err error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	identity = strings.TrimSpace(identity)
	ctx, observed := s.observer.Start(ctx, "tenant.hard_delete", nil, nil)
	defer observed.End(&err)
	if _, err = workspaceID(ctx); err != nil {
		return err
	}
	if identity == "" {
		return apperrors.InvalidInput("validation_error", "tenant identity is required")
	}
	if err = s.tenants.HardDelete(ctx, identity); err != nil {
		return err
	}
	observed.RecordStep("tenant.hard_delete.persisted", "tenant deleted", nil, zap.String("tenant_identity", identity))
	return nil
}

// ensureRootFolder returns the workspace root-tenants folder, creating it on
// first use. A concurrent creation is resolved by re-reading the root.
func (s *Tenant) ensureRootFolder(ctx context.Context, workspaceID uuid.UUID) (*entities.RFolder, error) {
	folder, err := s.folders.GetByIdentity(ctx, nil, entities.FolderEntityTypeTenants, entities.TenantRootFolderIdentity)
	if err == nil {
		return folder, nil
	}
	if apperrors.CodeOf(err) != "not_found" {
		return nil, err
	}
	root := &entities.RFolder{WorkspaceID: workspaceID, EntityType: entities.FolderEntityTypeTenants, Identity: entities.TenantRootFolderIdentity, DisplayName: entities.TenantRootFolderIdentity, IsRoot: true, IsSystem: true, Meta: map[string]any{}}
	folder, err = s.folders.Create(ctx, root)
	if err == nil {
		return folder, nil
	}
	if apperrors.CodeOf(err) != "identity_conflict" {
		return nil, err
	}
	return s.folders.GetByIdentity(ctx, nil, entities.FolderEntityTypeTenants, entities.TenantRootFolderIdentity)
}

// resolveFolderID returns root-tenants for a nil identity or resolves an
// explicit tenant folder identity within the current workspace.
func (s *Tenant) resolveFolderID(ctx context.Context, workspaceID uuid.UUID, identity *string, root *entities.RFolder) (*uuid.UUID, error) {
	if identity == nil {
		return &root.ID, nil
	}
	folder, err := s.resolveTenantFolder(ctx, workspaceID, *identity)
	if err != nil {
		return nil, err
	}
	return &folder.ID, nil
}

// resolveTenantFolder delegates public tenant folder identities to the shared,
// workspace-scoped relation resolver.
func (s *Tenant) resolveTenantFolder(ctx context.Context, workspaceID uuid.UUID, identity string) (*entities.RFolder, error) {
	return s.relations.ResolveFolder(ctx, workspaceID, identity, entities.FolderEntityTypeTenants, nil)
}

func (s *Tenant) toView(ctx context.Context, tenant *entities.RTenant) (*TenantView, error) {
	view := &TenantView{Tenant: tenant}
	if tenant == nil || tenant.FolderID == nil {
		return view, nil
	}
	folder, err := s.folders.GetByID(ctx, *tenant.FolderID)
	if err != nil {
		return nil, err
	}
	identity := folder.Identity
	view.FolderIdentity = &identity
	return view, nil
}

// workspaceID obtains the request workspace boundary and rejects unscoped use
// case calls before they can reach persistence.
func workspaceID(ctx context.Context) (uuid.UUID, error) {
	id, ok := entities.WorkspaceIDFromContext(ctx)
	if !ok {
		return uuid.Nil, apperrors.InvalidInput("workspace_required", "workspace scope is required")
	}
	return id, nil
}
