// Package relations содержит usecase-level resolver связей между domain
// entities. Он принимает публичные identity, использует только repository
// ports и не зависит от HTTP transport, Fiber или PostgreSQL.
package relations

import (
	"context"
	stderrors "errors"
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	apperrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/google/uuid"
)

// Resolver централизует разрешение project и folder identity в рамках
// workspace. Проверка scope, entity type и project принадлежности выполняется
// здесь, чтобы одинаковая логика не дублировалась в usecase.
type Resolver struct {
	projects ports.ProjectsRepository
	folders  ports.FoldersRepository
}

// NewResolver создаёт resolver связей поверх domain-oriented repository ports.
// Допускает nil для repository, который не нужен конкретному usecase: вызов
// метода, требующего отсутствующий repository, вернёт internal_error.
func NewResolver(projects ports.ProjectsRepository, folders ports.FoldersRepository) *Resolver {
	return &Resolver{projects: projects, folders: folders}
}

// ResolveProject разрешает project identity в техническую сущность текущего workspace.
//
// Параметры:
//
//	ctx         - контекст с обязательным WorkspaceScope;
//	workspaceID - workspace, который обязан совпадать со scope из ctx;
//	identity    - публичный project identity.
//
// Что делает функция:
//
//	Проверяет workspace scope, нормализует и валидирует identity, получает
//	Project через repository и убеждается, что полученная сущность принадлежит
//	текущему workspace. Global lookup для чужого workspace не выполняется.
//
// Возвращаемые значения:
//
//	*entities.RProject - resolved project с техническим UUID;
//	error              - validation_error, workspace_required,
//	                     workspace_scope_mismatch, project_not_found,
//	                     project_workspace_mismatch или repository error.
func (r *Resolver) ResolveProject(ctx context.Context, workspaceID uuid.UUID, identity string) (*entities.RProject, error) {
	if err := requireWorkspace(ctx, workspaceID); err != nil {
		return nil, err
	}
	identity = strings.TrimSpace(identity)
	if identity == "" {
		return nil, apperrors.InvalidInput("validation_error", "project identity is required")
	}
	if r == nil || r.projects == nil {
		return nil, apperrors.Internal("internal_error", "project relation resolver is unavailable")
	}
	project, err := r.projects.GetByIdentity(ctx, identity)
	if stderrors.Is(err, apperrors.ErrNotFound) {
		return nil, apperrors.NotFound("project_not_found", "project not found")
	}
	if err != nil {
		return nil, err
	}
	if project == nil || project.WorkspaceID != workspaceID {
		return nil, apperrors.InvalidInput("project_workspace_mismatch", "project must belong to the current workspace")
	}
	return project, nil
}

// ResolveProjectFromContext разрешает project identity в workspace из ctx.
//
// Что делает функция:
//
//	Извлекает workspaceID из проверенного request context и передаёт его в
//	ResolveProject. Это основной вариант для HTTP-facing usecase: workspace не
//	должен приходить из public input отдельно.
//
// Возвращаемые значения:
//
//	*entities.RProject - resolved project с техническим UUID;
//	error              - workspace_required или ошибка ResolveProject.
func (r *Resolver) ResolveProjectFromContext(ctx context.Context, identity string) (*entities.RProject, error) {
	workspaceID, err := workspaceIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return r.ResolveProject(ctx, workspaceID, identity)
}

// ResolveProjectID разрешает project identity в технический UUID.
//
// Что делает функция:
//
//	Вызывает ResolveProject с явным workspace scope и возвращает только ID
//	найденной сущности для repository/usecase операций.
//
// Возвращаемые значения:
//
//	uuid.UUID - технический UUID проекта;
//	error     - ошибки scope, validation и not found из ResolveProject.
func (r *Resolver) ResolveProjectID(ctx context.Context, workspaceID uuid.UUID, identity string) (uuid.UUID, error) {
	project, err := r.ResolveProject(ctx, workspaceID, identity)
	if err != nil {
		return uuid.Nil, err
	}
	return project.ID, nil
}

// ResolveFolder разрешает folder identity в техническую сущность с проверкой workspace,
// expectedEntityType и optional project scope.
//
// Параметры:
//
//	ctx                - контекст с обязательным WorkspaceScope;
//	workspaceID        - workspace, который обязан совпадать со scope из ctx;
//	identity           - публичный folder identity;
//	expectedEntityType - ожидаемый domain type папки;
//	projectID          - optional UUID проекта, которому должна принадлежать папка.
//
// Что делает функция:
//
//	Проверяет workspace scope, нормализует и валидирует identity и entity type,
//	выполняет scoped lookup папки и сверяет её workspace, entity type и project.
//	Global lookup при not found не выполняется: caller не узнаёт, существует ли
//	такая identity в другом workspace, project или entity domain.
//
// Возвращаемые значения:
//
//	*entities.RFolder - resolved folder с техническим UUID;
//	error             - validation_error, workspace_required,
//	                    workspace_scope_mismatch, folder_not_found,
//	                    folder_workspace_mismatch, folder_entity_type_mismatch,
//	                    folder_project_mismatch или repository error.
func (r *Resolver) ResolveFolder(ctx context.Context, workspaceID uuid.UUID, identity string, expectedEntityType entities.FolderEntityType, projectID *uuid.UUID) (*entities.RFolder, error) {
	if err := requireWorkspace(ctx, workspaceID); err != nil {
		return nil, err
	}
	identity = strings.TrimSpace(identity)
	if identity == "" {
		return nil, apperrors.InvalidInput("validation_error", "folder identity is required")
	}
	if strings.TrimSpace(string(expectedEntityType)) == "" {
		return nil, apperrors.InvalidInput("validation_error", "folder entity type is required")
	}
	if r == nil || r.folders == nil {
		return nil, apperrors.Internal("internal_error", "folder relation resolver is unavailable")
	}
	folder, err := r.folders.GetByIdentity(ctx, projectID, expectedEntityType, identity)
	if stderrors.Is(err, apperrors.ErrNotFound) {
		return nil, apperrors.NotFound("folder_not_found", "folder not found")
	}
	if err != nil {
		return nil, err
	}
	if folder == nil || folder.WorkspaceID != workspaceID {
		return nil, apperrors.InvalidInput("folder_workspace_mismatch", "folder must belong to the current workspace")
	}
	if folder.EntityType != expectedEntityType {
		return nil, apperrors.InvalidInput("folder_entity_type_mismatch", "folder has an unexpected entity type")
	}
	if !sameOptionalUUID(folder.ProjectID, projectID) {
		return nil, apperrors.InvalidInput("folder_project_mismatch", "folder must belong to the resolved project")
	}
	return folder, nil
}

// ResolveFolderFromContext разрешает folder identity в workspace из ctx с
// проверкой entity type и optional project scope.
//
// Что делает функция:
//
//	Извлекает workspaceID из проверенного request context и передаёт его в
//	ResolveFolder вместе с правилами entity type и project scope.
//
// Возвращаемые значения:
//
//	*entities.RFolder - resolved folder с техническим UUID;
//	error             - workspace_required или ошибка ResolveFolder.
func (r *Resolver) ResolveFolderFromContext(ctx context.Context, identity string, expectedEntityType entities.FolderEntityType, projectID *uuid.UUID) (*entities.RFolder, error) {
	workspaceID, err := workspaceIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return r.ResolveFolder(ctx, workspaceID, identity, expectedEntityType, projectID)
}

// ResolveFolderID разрешает folder identity в технический UUID.
//
// Что делает функция:
//
//	Вызывает ResolveFolder с явным workspace scope и возвращает только ID
//	найденной папки для repository/usecase операций.
//
// Возвращаемые значения:
//
//	uuid.UUID - технический UUID папки;
//	error     - проверки и domain errors из ResolveFolder.
func (r *Resolver) ResolveFolderID(ctx context.Context, workspaceID uuid.UUID, identity string, expectedEntityType entities.FolderEntityType, projectID *uuid.UUID) (uuid.UUID, error) {
	folder, err := r.ResolveFolder(ctx, workspaceID, identity, expectedEntityType, projectID)
	if err != nil {
		return uuid.Nil, err
	}
	return folder.ID, nil
}

func requireWorkspace(ctx context.Context, workspaceID uuid.UUID) error {
	scopedWorkspaceID, err := workspaceIDFromContext(ctx)
	if err != nil {
		return err
	}
	if workspaceID == uuid.Nil || scopedWorkspaceID != workspaceID {
		return apperrors.InvalidInput("workspace_scope_mismatch", "relation workspace must match the request scope")
	}
	return nil
}

func workspaceIDFromContext(ctx context.Context) (uuid.UUID, error) {
	workspaceID, ok := entities.WorkspaceIDFromContext(ctx)
	if !ok {
		return uuid.Nil, apperrors.InvalidInput("workspace_required", "workspace scope is required")
	}
	return workspaceID, nil
}

func sameOptionalUUID(left, right *uuid.UUID) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}
