// Package workspace_state владеет атомарным export/import и восстановлением полного состояния workspace.
package workspace_state

import (
	"context"

	"github.com/endge-lab/service-backend/internal/config"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
)

const SchemaVersion = 1

var Collections = []string{"projects", "tenants", "environments", "folders", "types", "queries", "data-views", "compositions", "stores", "streams", "updates", "mocks", "components", "actions", "filters", "converters", "computations", "vocabs", "i18n-bundles", "auth-profiles", "navigations", "styles"}
var UnsupportedCollections = []string{"parameters", "legacyComponents", "componentsDSL", "componentsTable", "versions", "pages", "pageTemplates", "page-templates", "policies"}
var readOnlyFields = []string{"id", "type", "revision", "author", "createdBy", "updatedBy", "createdAt", "updatedAt", "deletedAt", "created_by", "updated_by", "state"}

// Repository задаёт необходимые координатору операции хранилища.
type Repository interface {
	ports.WorkspaceRepository
	ports.IntegrationRepository
	ports.DocumentRepository
	ports.RevisionRepository
	ports.CommitRepository
	ports.ReleaseRepository
	ports.PortableRepository
	ports.SnapshotRepository
}

// Coordinator координирует импорт и восстановление состояния рабочего пространства.
type Coordinator struct {
	repository          Repository
	tx                  ports.TxManager
	backupRetentionDays int
	artifacts           ports.ReleaseArtifactReader
}

// mutationBatchContextKey задаёт закрытый тип ключа контекста для пакета мутаций.
type mutationBatchContextKey struct{}

// NewCoordinator создаёт координатор операций над состоянием рабочего пространства.
func NewCoordinator(repository Repository, tx ports.TxManager, cfg *config.Config, artifacts ports.ReleaseArtifactReader) *Coordinator {
	return &Coordinator{repository: repository, tx: tx, backupRetentionDays: cfg.Snapshots.ImportBackupRetentionDays, artifacts: artifacts}
}

// actor возвращает текущего пользователя из контекста.
func actor(ctx context.Context) (entities.CurrentActor, error) {
	value, ok := entities.CurrentActorFromContext(ctx)
	if !ok || value.User == nil {
		return value, domainerrors.Unauthorized("auth.current_user_missing", "Current user is missing")
	}
	return value, nil
}

// access возвращает доступ пользователя к рабочему пространству.
func access(ctx context.Context) (entities.WorkspaceAccess, error) {
	value, ok := entities.WorkspaceAccessFromContext(ctx)
	if !ok {
		return value, domainerrors.InvalidInput("workspace_required", "Workspace context is required")
	}
	return value, nil
}

// canWrite проверяет право роли изменять рабочее пространство.
func canWrite(role string) bool {
	return role == "editor" || role == "admin" || role == "platform_admin"
}

// canAdmin проверяет административное право роли.
func canAdmin(role string) bool { return role == "admin" || role == "platform_admin" }
