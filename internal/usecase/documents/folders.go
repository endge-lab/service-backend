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
	identity := stringField(input, "folderIdentity")
	if kind == "folders" {
		entityType := entities.FolderEntityType(stringField(input, "entityType"))
		if entityType == "" {
			return nil, domainerrors.InvalidInput("folder_entity_type_required", "entityType is required")
		}
		identity = stringField(input, "parentIdentity")
		if identity == "" && !boolField(input, "isRoot") {
			identity = entities.RootFolderIdentity(entityType)
		}
		return s.documents.ResolveFolder(ctx, scope.Workspace.ID, identity, entityType)
	}
	if identity == "" {
		identity = entities.RootFolderIdentity(kind)
	}
	return s.documents.ResolveFolder(ctx, scope.Workspace.ID, identity, entities.FolderEntityType(kind))
}

// resolveDocumentFolder разрешает папку, указанную в документе.
func (s *Lifecycle) resolveDocumentFolder(ctx context.Context, scope entities.WorkspaceAccess, document entities.Document) (*string, error) {
	var data map[string]any
	_ = json.Unmarshal(document.Data, &data)
	if document.Type == "folders" {
		entityType := entities.FolderEntityType(stringField(data, "entityType"))
		parent := stringField(data, "parentIdentity")
		if parent == "" && !boolField(data, "isRoot") {
			parent = entities.RootFolderIdentity(entityType)
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
	if kind == "folders" {
		if entityType := stringField(input, "entityType"); entityType != "" {
			input["entityType"] = entities.FolderEntityType(entityType)
		}
	}
}

// replaceStructuredRelations обновляет структурированные связи документа.
func (s *Lifecycle) replaceStructuredRelations(ctx context.Context, document entities.Document) error {
	if document.Type != "projects" {
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
	return boolField(data, "isRoot") || document.ManagedBy == "system"
}
