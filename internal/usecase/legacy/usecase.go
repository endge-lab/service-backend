// Package legacy содержит временные миграционные сценарии, удаляемые после переходного периода.
package legacy

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/history"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
	"github.com/google/uuid"
)

const rebuildOperation = "legacy_workspace_folder_rebuild"

type workspaceLocker interface {
	LockWorkspaceSnapshot(context.Context, string) error
}

// RebuildWorkspaceFoldersResult описывает итог атомарного пересоздания Workspace-проекции.
type RebuildWorkspaceFoldersResult struct {
	FoldersDeleted    int
	FoldersCreated    int
	DocumentsRelinked int
}

// UseCase владеет временной миграцией папок из Frontend-проекции в Workspace-проекцию.
type UseCase struct {
	documents ports.DocumentRepository
	locker    workspaceLocker
	tx        ports.TxManager
	history   *history.Recorder
}

// NewUseCase создаёт Legacy-сценарий пересборки Workspace-папок.
func NewUseCase(documents ports.DocumentRepository, locker ports.SnapshotRepository, tx ports.TxManager, recorder *history.Recorder) *UseCase {
	return &UseCase{documents: documents, locker: locker, tx: tx, history: recorder}
}

// RebuildWorkspaceFoldersFromFrontend удаляет пользовательскую Workspace-иерархию,
// копирует реальные Frontend-папки и перепривязывает документы без изменения folderId.
func (s *UseCase) RebuildWorkspaceFoldersFromFrontend(ctx context.Context, confirmation string) (result RebuildWorkspaceFoldersResult, err error) {
	current, scope, err := shared.AdminContext(ctx)
	if err != nil {
		return result, err
	}
	if strings.TrimSpace(confirmation) != scope.Workspace.Identity {
		return result, domainerrors.InvalidInput("workspace_confirmation_invalid", "confirmation must match target workspace identity")
	}

	err = s.tx.WithinTransaction(ctx, func(txctx context.Context) error {
		if lockErr := s.locker.LockWorkspaceSnapshot(txctx, scope.Workspace.ID); lockErr != nil {
			return lockErr
		}

		folders, loadErr := s.documents.ListAllDocuments(txctx, scope.Workspace.ID, entities.CollectionFolders, false)
		if loadErr != nil {
			return loadErr
		}
		workspaceRoot := findWorkspaceRoot(folders)
		if workspaceRoot == nil {
			return domainerrors.Internal("workspace_root_folder_missing", "Workspace root folder is missing")
		}

		documents, loadErr := s.loadDocuments(txctx, scope.Workspace.ID)
		if loadErr != nil {
			return loadErr
		}

		txctx, err = s.history.BeginBatch(txctx, &scope.Workspace.ID, rebuildOperation, current.User.ID)
		if err != nil {
			return err
		}

		for _, folder := range folders {
			if !isWorkspaceFolder(folder) || folder.Identity == entities.WorkspaceRootFolderIdentity {
				continue
			}
			now := time.Now().UTC()
			next := folder
			next.Active = false
			next.DeletedAt = &now
			next.UpdatedBy = entities.Actor{ID: current.User.ID}
			deleted, deleteErr := s.documents.UpdateDocument(txctx, next, folder.Revision, parentID(folder, folders))
			if deleteErr != nil {
				return deleteErr
			}
			if _, deleteErr = s.history.RecordDocument(txctx, *deleted, "delete", nil); deleteErr != nil {
				return deleteErr
			}
			result.FoldersDeleted++
		}

		placements, createErr := s.createWorkspaceFolders(txctx, scope.Workspace.ID, current.User.ID, workspaceRoot.ID, folders, &result)
		if createErr != nil {
			return createErr
		}

		workspaceRootPlacement := folderPlacement{ID: workspaceRoot.ID, Identity: workspaceRoot.Identity}
		storePlacements := make(map[string]folderPlacement)
		for _, document := range documents {
			if document.Type == entities.CollectionStores {
				storePlacements[document.Identity] = frontendPlacement(document, placements, workspaceRootPlacement)
			}
		}

		for _, document := range documents {
			target := frontendPlacement(document, placements, workspaceRootPlacement)
			if document.Type == entities.CollectionUpdates {
				storeIdentity := stringField(folderData(document), "storeIdentity")
				if storeTarget := storePlacements[storeIdentity]; storeTarget.ID != "" {
					target = storeTarget
				}
			}
			if document.WorkspaceFolderIdentity != nil && *document.WorkspaceFolderIdentity == target.Identity {
				continue
			}
			updated, updateErr := s.documents.UpdateDocumentWorkspaceFolder(
				txctx, scope.Workspace.ID, document.Type, document.Identity, target.ID, current.User.ID, document.Revision,
			)
			if updateErr != nil {
				return updateErr
			}
			if _, updateErr = s.history.RecordDocument(txctx, *updated, "update", nil); updateErr != nil {
				return updateErr
			}
			result.DocumentsRelinked++
		}
		return nil
	})
	if err != nil {
		return RebuildWorkspaceFoldersResult{}, shared.MapConflict(err)
	}
	return result, nil
}

func (s *UseCase) loadDocuments(ctx context.Context, workspaceID string) ([]entities.Document, error) {
	result := []entities.Document{}
	for _, collection := range entities.DocumentCollections {
		if collection == entities.CollectionFolders {
			continue
		}
		items, err := s.documents.ListAllDocuments(ctx, workspaceID, collection, true)
		if err != nil {
			return nil, err
		}
		result = append(result, items...)
	}
	return result, nil
}

type folderAppearance struct {
	Identity    string
	DisplayName string
	Icon        string
	Color       string
}

// frontendRoots повторяет реальные корни, видимые в Frontend-метамодели.
// Виртуальные Workspace, Events, Integrations и facets здесь намеренно отсутствуют.
var frontendRoots = []folderAppearance{
	{Identity: "root-tenants", DisplayName: "Тенанты", Icon: "Building2", Color: "#10b981"},
	{Identity: "root-projects", DisplayName: "Проекты", Icon: "Briefcase", Color: "#0ea5e9"},
	{Identity: "root-environments", DisplayName: "Окружения", Icon: "ServerCog", Color: "#84cc16"},
	{Identity: "root-stores", DisplayName: "Хранилища", Icon: "Database", Color: "#10b981"},
	{Identity: "root-vocabs", DisplayName: "Словари", Icon: "BookOpen", Color: "#14b8a6"},
	{Identity: "root-mocks", DisplayName: "Тестовые данные", Icon: "Braces", Color: "#8b5a2b"},
	{Identity: "root-types", DisplayName: "Типы", Icon: "Type", Color: "#3b82f6"},
	{Identity: "root-queries", DisplayName: "Обмен данными", Icon: "Send", Color: "#f97316"},
	{Identity: "root-data-views", DisplayName: "Представления", Icon: "GitBranch", Color: "#06b6d4"},
	{Identity: "root-compositions", DisplayName: "Композиции", Icon: "Network", Color: "#8b5cf6"},
	{Identity: "root-components", DisplayName: "Компоненты", Icon: "Puzzle", Color: "#3b82f6"},
	{Identity: "root-actions", DisplayName: "Действия", Icon: "Zap", Color: "#f59e0b"},
	{Identity: "root-filters", DisplayName: "Фильтры", Icon: "Filter", Color: "#f43f5e"},
	{Identity: "root-converters", DisplayName: "Конвертеры", Icon: "ArrowLeftRight", Color: "#06b6d4"},
	{Identity: "root-computations", DisplayName: "Вычисления", Icon: "SquareFunction", Color: "#f97316"},
	{Identity: "root-i18n-bundles", DisplayName: "Словари переводов", Icon: "Languages", Color: "#f59e0b"},
	{Identity: "root-auth-profiles", DisplayName: "Профили аутентификации", Icon: "KeyRound", Color: "#0ea5e9"},
	{Identity: "root-simulations", DisplayName: "Симуляции", Icon: "FlaskConical", Color: "#ec4899"},
	{Identity: "root-navigations", DisplayName: "Навигация", Icon: "Route", Color: "#22d3ee"},
	{Identity: "root-styles", DisplayName: "Стили", Icon: "Palette", Color: "#d946ef"},
}

type folderPlacement struct {
	ID       string
	Identity string
}

func (s *UseCase) createWorkspaceFolders(ctx context.Context, workspaceID, actorID, workspaceRootID string, folders []entities.Document, result *RebuildWorkspaceFoldersResult) (map[string]folderPlacement, error) {
	sources := make(map[string]entities.Document, len(folders))
	children := make(map[string][]entities.Document)
	for _, folder := range folders {
		if !isCollectionFolder(folder) {
			continue
		}
		sources[folder.Identity] = folder
		if folder.FolderIdentity != nil {
			children[*folder.FolderIdentity] = append(children[*folder.FolderIdentity], folder)
		}
	}
	for parent := range children {
		sort.Slice(children[parent], func(i, j int) bool {
			if children[parent][i].DisplayName == children[parent][j].DisplayName {
				return children[parent][i].Identity < children[parent][j].Identity
			}
			return children[parent][i].DisplayName < children[parent][j].DisplayName
		})
	}

	placements := make(map[string]folderPlacement)
	for _, root := range frontendRoots {
		source := sources[root.Identity]
		created, err := s.createFolder(ctx, workspaceID, actorID, workspaceRootID, root.DisplayName, description(source), root.Icon, root.Color)
		if err != nil {
			return nil, err
		}
		placements[root.Identity] = folderPlacement{ID: created.ID, Identity: created.Identity}
		result.FoldersCreated++
		if err = s.createDescendants(ctx, workspaceID, actorID, root.Identity, created.ID, children, placements, result); err != nil {
			return nil, err
		}
	}
	return placements, nil
}

func (s *UseCase) createDescendants(ctx context.Context, workspaceID, actorID, sourceParent, targetParentID string, children map[string][]entities.Document, placements map[string]folderPlacement, result *RebuildWorkspaceFoldersResult) error {
	if sourceParent == "" {
		return nil
	}
	for _, source := range children[sourceParent] {
		data := folderData(source)
		icon, _ := data["icon"].(string)
		color, _ := data["color"].(string)
		if strings.TrimSpace(icon) == "" {
			icon = "Folder"
		}
		if strings.TrimSpace(color) == "" {
			color = "#eab308"
		}
		created, err := s.createFolder(ctx, workspaceID, actorID, targetParentID, source.DisplayName, source.Description, icon, strings.ToLower(color))
		if err != nil {
			return err
		}
		placements[source.Identity] = folderPlacement{ID: created.ID, Identity: created.Identity}
		result.FoldersCreated++
		if err = s.createDescendants(ctx, workspaceID, actorID, source.Identity, created.ID, children, placements, result); err != nil {
			return err
		}
	}
	return nil
}

func (s *UseCase) createFolder(ctx context.Context, workspaceID, actorID, parentID, displayName string, description *string, icon, color string) (*entities.Document, error) {
	identity := "legacy-workspace-" + uuid.NewString()
	data, _ := json.Marshal(map[string]any{
		"scope":          entities.FolderScopeWorkspace,
		"entityType":     nil,
		"parentIdentity": nil,
		"isRoot":         false,
		"icon":           icon,
		"color":          color,
	})
	document := entities.Document{
		ID: uuid.NewString(), WorkspaceID: workspaceID, Type: entities.CollectionFolders,
		Identity: identity, DisplayName: displayName, Description: description,
		ManagedBy: "user", Meta: json.RawMessage(`{}`), Data: data, Active: true, Revision: 1,
		CreatedBy: entities.Actor{ID: actorID}, UpdatedBy: entities.Actor{ID: actorID},
	}
	created, err := s.documents.InsertDocument(ctx, document, &parentID)
	if err != nil {
		return nil, err
	}
	if _, err = s.history.RecordDocument(ctx, *created, "create", nil); err != nil {
		return nil, err
	}
	return created, nil
}

func findWorkspaceRoot(folders []entities.Document) *entities.Document {
	for i := range folders {
		if folders[i].Identity == entities.WorkspaceRootFolderIdentity && isWorkspaceFolder(folders[i]) {
			return &folders[i]
		}
	}
	return nil
}

func isWorkspaceFolder(folder entities.Document) bool {
	return stringField(folderData(folder), "scope") == entities.FolderScopeWorkspace
}

func isCollectionFolder(folder entities.Document) bool {
	return stringField(folderData(folder), "scope") == entities.FolderScopeCollection
}

func folderData(folder entities.Document) map[string]any {
	value := map[string]any{}
	_ = json.Unmarshal(folder.Data, &value)
	return value
}

func stringField(value map[string]any, key string) string {
	text, _ := value[key].(string)
	return strings.TrimSpace(text)
}

func description(folder entities.Document) *string {
	if folder.Identity == "" {
		return nil
	}
	return folder.Description
}

func parentID(folder entities.Document, folders []entities.Document) *string {
	if folder.FolderIdentity == nil {
		return nil
	}
	for i := range folders {
		if folders[i].Identity == *folder.FolderIdentity {
			return &folders[i].ID
		}
	}
	return nil
}

func targetFrontendFolder(document entities.Document) string {
	// Updates во Frontend принадлежат Store и не имеют отдельного видимого корня.
	// В Workspace-проекции группируем их под скопированным корнем хранилищ.
	if document.Type == entities.CollectionUpdates {
		return entities.RootFolderIdentity(entities.CollectionStores)
	}
	if document.Type == entities.CollectionConfigurations {
		return ""
	}
	if document.FolderIdentity != nil && strings.TrimSpace(*document.FolderIdentity) != "" {
		return strings.TrimSpace(*document.FolderIdentity)
	}
	return entities.RootFolderIdentity(document.Type)
}

func frontendPlacement(document entities.Document, placements map[string]folderPlacement, fallback folderPlacement) folderPlacement {
	if target := placements[targetFrontendFolder(document)]; target.ID != "" {
		return target
	}
	return fallback
}
