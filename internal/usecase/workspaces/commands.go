package workspaces

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/documents"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
	"github.com/google/uuid"
)

// Create создаёт рабочее пространство и его начальное состояние.
func (s *UseCase) Create(ctx context.Context, input CreateInput) (result *entities.Workspace, err error) {
	current, err := shared.Actor(ctx)
	if err != nil {
		return nil, err
	}
	if !current.PlatformAdmin {
		return nil, domainerrors.Forbidden("platform_admin_required", "Platform Admin role is required")
	}
	values, err := inputValues(input)
	if err != nil {
		return nil, err
	}
	if err = shared.RejectReadOnly(values); err != nil {
		return nil, err
	}
	if err = shared.ValidateSecrets(values); err != nil {
		return nil, err
	}
	identity, displayName := workspaceText(values, "identity"), workspaceText(values, "displayName")
	if err = validateWorkspaceIdentity(identity); err != nil {
		return nil, err
	}
	if displayName == "" {
		return nil, domainerrors.InvalidInput("display_name_required", "displayName is required")
	}
	value := entities.Workspace{ID: uuid.NewString(), Identity: identity, DisplayName: displayName, Description: workspaceOptional(values, "description"), DataMode: workspaceDefault(workspaceText(values, "dataMode"), "development"), Configuration: workspaceJSON(values["configuration"], `{}`), Meta: workspaceJSON(values["meta"], `{}`), Active: workspaceBool(values, "active", true), Revision: 1, CreatedBy: entities.Actor{ID: current.User.ID}, UpdatedBy: entities.Actor{ID: current.User.ID}}
	err = s.tx.WithinTransaction(ctx, func(txctx context.Context) error {
		created, txErr := s.workspaces.CreateWorkspace(txctx, value, current.User.ID)
		if txErr != nil {
			return txErr
		}
		result = created
		bindings, txErr := workspaceIntegrations(values)
		if txErr != nil {
			return txErr
		}
		if txErr = s.workspaces.ReplaceWorkspaceIntegrations(txctx, created.ID, bindings, current.User.ID); txErr != nil {
			return txErr
		}
		txctx, txErr = s.history.BeginBatch(txctx, &created.ID, "create", current.User.ID)
		if txErr != nil {
			return txErr
		}
		if txErr = s.history.RecordWorkspace(txctx, *created, "create"); txErr != nil {
			return txErr
		}
		for _, kind := range documents.Collections {
			if kind == "folders" {
				continue
			}
			root := entities.Document{ID: uuid.NewString(), WorkspaceID: created.ID, Type: "folders", Identity: "root-" + kind, DisplayName: "Root " + kind, ManagedBy: "system", Meta: json.RawMessage(`{}`), Data: workspaceJSON(map[string]any{"entityType": kind, "isRoot": true}, `{}`), Active: true, Revision: 1, CreatedBy: entities.Actor{ID: current.User.ID}, UpdatedBy: entities.Actor{ID: current.User.ID}}
			createdRoot, insertErr := s.documents.InsertDocument(txctx, root, nil)
			if insertErr != nil {
				return insertErr
			}
			if _, insertErr = s.history.RecordDocument(txctx, *createdRoot, "create", nil); insertErr != nil {
				return insertErr
			}
		}
		pending, txErr := s.commits.PendingRevisions(txctx, created.ID, 0)
		if txErr != nil {
			return txErr
		}
		head := int64(0)
		for _, revision := range pending {
			if revision.WorkspaceSequence != nil && *revision.WorkspaceSequence > head {
				head = *revision.WorkspaceSequence
			}
		}
		commit, txErr := s.commits.CreateCommit(txctx, entities.Commit{ID: uuid.NewString(), WorkspaceID: created.ID, BaseSequence: 0, HeadSequence: head, Message: "Initial workspace state", RevisionPolicy: "preserve", Operation: "bootstrap", CreatedBy: entities.Actor{ID: current.User.ID}}, workspaceCommitChanges(pending))
		if txErr != nil {
			return txErr
		}
		ids := make([]string, 0, len(pending))
		for _, revision := range pending {
			ids = append(ids, revision.ID)
		}
		return s.commits.AttachRevisionsToCommit(txctx, commit.ID, ids)
	})
	return result, shared.MapConflict(err)
}

// Patch частично обновляет рабочее пространство с проверкой ожидаемой ревизии.
func (s *UseCase) Patch(ctx context.Context, identity string, input PatchInput, expected int) (result *entities.Workspace, err error) {
	current, err := shared.Actor(ctx)
	if err != nil {
		return nil, err
	}
	scope, err := s.Authorize(ctx, identity)
	if err != nil {
		return nil, err
	}
	if !shared.CanAdmin(scope.Role) {
		return nil, domainerrors.Forbidden("workspace_admin_required", "Workspace Admin role is required")
	}
	if expected <= 0 {
		return nil, shared.PreconditionRequired()
	}
	patch, err := input.values()
	if err != nil {
		return nil, err
	}
	if err = shared.RejectReadOnly(patch); err != nil {
		return nil, err
	}
	if err = shared.ValidateSecrets(patch); err != nil {
		return nil, err
	}
	if scope.Workspace.Revision != expected {
		return nil, shared.RevisionConflict()
	}
	next := applyWorkspacePatch(scope.Workspace, patch)
	if err = validateWorkspaceIdentity(next.Identity); err != nil {
		return nil, err
	}
	if strings.TrimSpace(next.DisplayName) == "" {
		return nil, domainerrors.InvalidInput("display_name_required", "displayName is required")
	}
	contentChanged := workspaceDigest(scope.Workspace) != workspaceDigest(next)
	bindings, bindingsPresent, err := patchedWorkspaceIntegrations(patch)
	if err != nil {
		return nil, err
	}
	bindingsChanged := false
	if bindingsPresent {
		currentBindings, listErr := s.workspaces.ListWorkspaceIntegrations(ctx, scope.Workspace.ID)
		if listErr != nil {
			return nil, listErr
		}
		bindingsChanged = genericDigest(currentBindings) != genericDigest(bindings)
	}
	if !contentChanged && !bindingsChanged {
		return &scope.Workspace, nil
	}
	err = s.tx.WithinTransaction(ctx, func(txctx context.Context) error {
		updated, txErr := s.workspaces.UpdateWorkspace(txctx, identity, patch, expected, current.User.ID)
		if txErr != nil {
			return txErr
		}
		if bindingsPresent {
			if txErr = s.workspaces.ReplaceWorkspaceIntegrations(txctx, updated.ID, bindings, current.User.ID); txErr != nil {
				return txErr
			}
		}
		if txErr = s.history.RecordWorkspace(txctx, *updated, "update"); txErr != nil {
			return txErr
		}
		result = updated
		return nil
	})
	return result, shared.MapConflict(err)
}

// applyWorkspacePatch применяет частичное обновление к рабочему пространству.
func applyWorkspacePatch(workspace entities.Workspace, patch map[string]any) entities.Workspace {
	if value, ok := patch["identity"].(string); ok {
		workspace.Identity = strings.TrimSpace(value)
	}
	if value, ok := patch["displayName"].(string); ok {
		workspace.DisplayName = value
	}
	if _, ok := patch["description"]; ok {
		workspace.Description = workspaceOptional(patch, "description")
	}
	if value, ok := patch["dataMode"].(string); ok {
		workspace.DataMode = value
	}
	if value, ok := patch["configuration"]; ok {
		workspace.Configuration = workspaceJSON(value, `{}`)
	}
	if value, ok := patch["meta"]; ok {
		workspace.Meta = workspaceJSON(value, `{}`)
	}
	if value, ok := patch["active"].(bool); ok {
		workspace.Active = value
	}
	return workspace
}

// workspaceCommitChanges формирует изменения коммита рабочего пространства.
func workspaceCommitChanges(revisions []entities.Revision) []entities.CommitChange {
	groups, order := map[string][]entities.Revision{}, []string{}
	for _, revision := range revisions {
		key := revision.DocumentType + ":" + revision.DocumentID
		if _, exists := groups[key]; !exists {
			order = append(order, key)
		}
		groups[key] = append(groups[key], revision)
	}
	result := make([]entities.CommitChange, 0, len(order))
	for _, key := range order {
		items := groups[key]
		first, last := items[0], items[len(items)-1]
		result = append(result, entities.CommitChange{DocumentType: last.DocumentType, DocumentID: last.DocumentID, BeforeRevisionID: first.ParentRevisionID, AfterRevisionID: &last.ID, Operation: last.Operation})
	}
	return result
}

// workspaceIntegrations загружает интеграции рабочего пространства для снимка.
func workspaceIntegrations(values map[string]any) ([]map[string]any, error) {
	result, _, err := patchedWorkspaceIntegrations(values)
	return result, err
}

// patchedWorkspaceIntegrations применяет обновление интеграций к снимку рабочего пространства.
func patchedWorkspaceIntegrations(values map[string]any) ([]map[string]any, bool, error) {
	value, exists := values["installedIntegrations"]
	if !exists || value == nil {
		return nil, exists, nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, true, domainerrors.InvalidInput("installed_integrations_invalid", "installedIntegrations is invalid")
	}
	var result []map[string]any
	if err = json.Unmarshal(raw, &result); err != nil {
		return nil, true, domainerrors.InvalidInput("installed_integrations_invalid", "installedIntegrations must be an array")
	}
	for _, item := range result {
		if _, ok := item["configuration"]; !ok {
			item["configuration"] = map[string]any{}
		}
	}
	return result, true, nil
}

// validateWorkspaceIdentity проверяет identity рабочего пространства.
func validateWorkspaceIdentity(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return domainerrors.InvalidInput("identity_required", "identity is required")
	}
	if len(value) > 160 {
		return domainerrors.InvalidInput("identity_too_long", "identity must not exceed 160 characters")
	}
	return nil
}

// workspaceText извлекает текстовое поле рабочего пространства.
func workspaceText(values map[string]any, key string) string {
	value, _ := values[key].(string)
	return strings.TrimSpace(value)
}

// workspaceOptional извлекает необязательное текстовое поле рабочего пространства.
func workspaceOptional(values map[string]any, key string) *string {
	value, ok := values[key].(string)
	value = strings.TrimSpace(value)
	if !ok || value == "" {
		return nil
	}
	return &value
}

// workspaceDefault возвращает строковое поле рабочего пространства или значение по умолчанию.
func workspaceDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

// workspaceBool извлекает логическое поле рабочего пространства.
func workspaceBool(values map[string]any, key string, fallback bool) bool {
	value, ok := values[key].(bool)
	if !ok {
		return fallback
	}
	return value
}

// workspaceJSON извлекает JSON-поле рабочего пространства.
func workspaceJSON(value any, fallback string) json.RawMessage {
	if value == nil {
		return json.RawMessage(fallback)
	}
	raw, _ := json.Marshal(value)
	return raw
}

// genericDigest вычисляет контрольную сумму произвольного значения.
func genericDigest(value any) string {
	raw, _ := json.Marshal(value)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// workspaceDigest вычисляет контрольную сумму рабочего пространства.
func workspaceDigest(value entities.Workspace) string {
	return genericDigest(map[string]any{"identity": value.Identity, "displayName": value.DisplayName, "description": value.Description, "dataMode": value.DataMode, "configuration": value.Configuration, "meta": value.Meta, "active": value.Active})
}
