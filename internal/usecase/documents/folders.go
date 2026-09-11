package documents

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
)

// resolveFolder разрешает identity папки в её внутренний идентификатор.
func (s *Lifecycle) resolveFolder(ctx context.Context, scope entities.WorkspaceAccess, kind string, input map[string]any) (*string, error) {
	if kind == entities.CollectionConfigurations {
		return nil, nil
	}
	identity := stringField(input, "folderIdentity")
	if kind == entities.CollectionFolders {
		folderScope := defaultString(stringField(input, "scope"), entities.FolderScopeCollection)
		entityType := entities.FolderEntityType(stringField(input, "entityType"))
		identity = stringField(input, "parentIdentity")
		if identity == "" && !isRoot(input) {
			if folderScope == entities.FolderScopeWorkspace {
				identity = entities.WorkspaceRootFolderIdentity
			} else {
				identity = entities.RootFolderIdentity(entityType)
			}
		}
		return s.documents.ResolveFolder(ctx, scope.Workspace.ID, identity, entityType)
	}
	if identity == "" {
		identity = entities.RootFolderIdentity(kind)
	}
	return s.documents.ResolveFolder(ctx, scope.Workspace.ID, identity, entities.FolderEntityType(kind))
}

// resolveDocumentWorkspaceFolder проверяет вторую, общую для Workspace, папку документа.
func (s *Lifecycle) resolveDocumentWorkspaceFolder(ctx context.Context, scope entities.WorkspaceAccess, document entities.Document) error {
	if document.Type == entities.CollectionFolders {
		return nil
	}
	identity := entities.WorkspaceRootFolderIdentity
	if document.WorkspaceFolderIdentity != nil && strings.TrimSpace(*document.WorkspaceFolderIdentity) != "" {
		identity = strings.TrimSpace(*document.WorkspaceFolderIdentity)
	}
	_, err := s.documents.ResolveFolder(ctx, scope.Workspace.ID, identity, "")
	return err
}

// resolveDocumentFolder разрешает папку, указанную в документе.
func (s *Lifecycle) resolveDocumentFolder(ctx context.Context, scope entities.WorkspaceAccess, document entities.Document) (*string, error) {
	if document.Type == entities.CollectionConfigurations {
		return nil, nil
	}
	var data map[string]any
	_ = json.Unmarshal(document.Data, &data)
	if document.Type == entities.CollectionFolders {
		entityType := entities.FolderEntityType(stringField(data, "entityType"))
		folderScope := defaultString(stringField(data, "scope"), entities.FolderScopeCollection)
		parent := stringField(data, "parentIdentity")
		if parent == "" && !isRoot(data) {
			if folderScope == entities.FolderScopeWorkspace {
				parent = entities.WorkspaceRootFolderIdentity
			} else {
				parent = entities.RootFolderIdentity(entityType)
			}
		}
		return s.documents.ResolveFolder(ctx, scope.Workspace.ID, parent, entityType)
	}
	identity := ""
	if document.FolderIdentity != nil {
		identity = *document.FolderIdentity
	}
	if identity == "" {
		identity = entities.RootFolderIdentity(document.Type)
	}
	return s.documents.ResolveFolder(ctx, scope.Workspace.ID, identity, entities.FolderEntityType(document.Type))
}

// normalizeFolderInput приводит тип папок к общей физической секции коллекции.
func normalizeFolderInput(kind string, input map[string]any) {
	if kind == entities.CollectionFolders {
		if scope := stringField(input, "scope"); scope != "" {
			input["scope"] = scope
		}
		if entityType := stringField(input, "entityType"); entityType != "" {
			input["entityType"] = entities.FolderEntityType(entityType)
		}
	}
}

// validateFolderScopePatch запрещает превращать существующую папку в папку другой проекции.
func validateFolderScopePatch(existing entities.Document, patch map[string]any) error {
	var data map[string]any
	_ = json.Unmarshal(existing.Data, &data)
	currentScope := defaultString(stringField(data, "scope"), entities.FolderScopeCollection)
	currentEntityType := entities.FolderEntityType(stringField(data, "entityType"))
	if value, exists := patch["scope"]; exists {
		patchedScope, _ := value.(string)
		if defaultString(strings.TrimSpace(patchedScope), entities.FolderScopeCollection) != currentScope {
			return domainerrors.Conflict("folder_scope_immutable", "Folder scope cannot be changed")
		}
	}
	if value, exists := patch["entityType"]; exists {
		patchedEntityType := ""
		if value != nil {
			patchedEntityType = entities.FolderEntityType(stringField(map[string]any{"entityType": value}, "entityType"))
		}
		if patchedEntityType != currentEntityType {
			return domainerrors.Conflict("folder_entity_type_immutable", "Folder entityType cannot be changed")
		}
	}
	return nil
}

// ensureWorkspaceFolderInput задаёт скрытый Workspace-корень для нового документа.
func ensureWorkspaceFolderInput(kind string, input map[string]any) {
	if kind == entities.CollectionFolders {
		return
	}
	if stringField(input, "workspaceFolderIdentity") == "" {
		input["workspaceFolderIdentity"] = entities.WorkspaceRootFolderIdentity
	}
}

// replaceStructuredRelations обновляет структурированные связи документа.
func (s *Lifecycle) replaceStructuredRelations(ctx context.Context, document entities.Document) error {
	if document.Type != entities.CollectionProjects {
		return nil
	}
	var data map[string]any
	if err := json.Unmarshal(document.Data, &data); err != nil {
		return domainerrors.InvalidInput("document_data_invalid", "Document data is invalid")
	}
	environments := relationIdentities(data["allowedEnvironments"])
	if len(environments) == 0 {
		environments = relationIdentities(data["allowedEnvironmentIdentities"])
	}
	if err := s.documents.ReplaceProjectEnvironments(ctx, document, environments); err != nil {
		if strings.Contains(err.Error(), "relation target") {
			return domainerrors.InvalidInput("relation_target_not_found", err.Error())
		}
		return err
	}
	return nil
}

// relationIdentities извлекает identity связанных документов.
func relationIdentities(value any) []string {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	seen, result := map[string]bool{}, []string{}
	for _, item := range items {
		identity := ""
		switch typed := item.(type) {
		case string:
			identity = strings.TrimSpace(typed)
		case map[string]any:
			identity = stringField(typed, "identity")
		}
		if identity != "" && !seen[identity] {
			seen[identity] = true
			result = append(result, identity)
		}
	}
	return result
}

// isSystemFolder определяет системную папку по владельцу управления.
func isSystemFolder(document entities.Document) bool {
	var data map[string]any
	_ = json.Unmarshal(document.Data, &data)
	return isRoot(data) || document.ManagedBy == entities.ManagedBySystem
}
