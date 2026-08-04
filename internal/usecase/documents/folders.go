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
		entityType := stringField(input, "entityType")
		if entityType == "" {
			return nil, domainerrors.InvalidInput("folder_entity_type_required", "entityType is required")
		}
		identity = stringField(input, "parentIdentity")
		if identity == "" && !boolField(input, "isRoot") {
			identity = "root-" + entityType
		}
		return s.documents.ResolveFolder(ctx, scope.Workspace.ID, identity, entityType)
	}
	if identity == "" {
		identity = "root-" + kind
	}
	return s.documents.ResolveFolder(ctx, scope.Workspace.ID, identity, kind)
}

// resolveDocumentFolder разрешает папку, указанную в документе.
func (s *Lifecycle) resolveDocumentFolder(ctx context.Context, scope entities.WorkspaceAccess, document entities.Document) (*string, error) {
	var data map[string]any
	_ = json.Unmarshal(document.Data, &data)
	if document.Type == "folders" {
		parent := stringField(data, "parentIdentity")
		if parent == "" && !boolField(data, "isRoot") {
			parent = "root-" + stringField(data, "entityType")
		}
		return s.documents.ResolveFolder(ctx, scope.Workspace.ID, parent, stringField(data, "entityType"))
	}
	identity := ""
	if document.FolderIdentity != nil {
		identity = *document.FolderIdentity
	}
	if identity == "" {
		identity = "root-" + document.Type
	}
	return s.documents.ResolveFolder(ctx, scope.Workspace.ID, identity, document.Type)
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
