package workspace_state

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/google/uuid"
)

// writeContext получает актёра и область записи из контекста запроса.
func (s *Coordinator) writeContext(ctx context.Context) (entities.CurrentActor, entities.WorkspaceAccess, error) {
	current, err := actor(ctx)
	if err != nil {
		return current, entities.WorkspaceAccess{}, err
	}
	scope, err := access(ctx)
	if err != nil {
		return current, scope, err
	}
	if !canWrite(scope.Role) {
		return current, scope, domainerrors.Forbidden("workspace_editor_required", "Workspace Editor role is required")
	}
	return current, scope, nil
}

// recordRevision записывает ревизию документа в текущей транзакции.
func (s *Coordinator) recordRevision(ctx context.Context, doc entities.Document, operation string, restored *string) (*entities.Revision, error) {
	sequence, err := s.repository.NextWorkspaceSequence(ctx, doc.WorkspaceID)
	if err != nil {
		return nil, err
	}
	batch, err := s.mutationBatch(ctx, &doc.WorkspaceID, operation, doc.UpdatedBy.ID)
	if err != nil {
		return nil, err
	}
	latest, err := s.repository.LatestRevision(ctx, &doc.WorkspaceID, doc.Type, doc.ID)
	var parent *string
	if err == nil {
		parent = &latest.ID
	} else if !errors.Is(err, ports.ErrNotFound) {
		return nil, err
	}
	snapshot := mustJSON(doc)
	value := entities.Revision{ID: uuid.NewString(), WorkspaceID: doc.WorkspaceID, DocumentType: doc.Type, DocumentID: doc.ID, DocumentIdentity: doc.Identity, RevisionNumber: doc.Revision, WorkspaceSequence: &sequence, Operation: operation, ParentRevisionID: parent, RestoredFromRevisionID: restored, MutationBatchID: batch, SnapshotVersion: SchemaVersion, Snapshot: snapshot, Checksum: checksum(snapshot), CreatedBy: doc.UpdatedBy}
	return s.repository.InsertRevision(ctx, value)
}

// recordWorkspaceRevision записывает ревизию рабочего пространства.
func (s *Coordinator) recordWorkspaceRevision(ctx context.Context, workspace entities.Workspace, operation string) (*entities.Revision, error) {
	sequence, err := s.repository.NextWorkspaceSequence(ctx, workspace.ID)
	if err != nil {
		return nil, err
	}
	batch, err := s.mutationBatch(ctx, &workspace.ID, operation, workspace.UpdatedBy.ID)
	if err != nil {
		return nil, err
	}
	latest, err := s.repository.LatestRevision(ctx, &workspace.ID, "workspaces", workspace.ID)
	var parent *string
	if err == nil {
		parent = &latest.ID
	} else if !errors.Is(err, ports.ErrNotFound) {
		return nil, err
	}
	snapshot := mustJSON(workspace)
	value := entities.Revision{ID: uuid.NewString(), WorkspaceID: workspace.ID, DocumentType: "workspaces", DocumentID: workspace.ID, DocumentIdentity: workspace.Identity, RevisionNumber: workspace.Revision, WorkspaceSequence: &sequence, Operation: operation, ParentRevisionID: parent, MutationBatchID: batch, SnapshotVersion: SchemaVersion, Snapshot: snapshot, Checksum: checksum(snapshot), CreatedBy: workspace.UpdatedBy}
	return s.repository.InsertRevision(ctx, value)
}

// mutationBatch создаёт или переиспользует пакет связанных изменений.
func (s *Coordinator) mutationBatch(ctx context.Context, workspaceID *string, operation, actorID string) (string, error) {
	if batch, ok := ctx.Value(mutationBatchContextKey{}).(string); ok && batch != "" {
		return batch, nil
	}
	return s.repository.CreateMutationBatch(ctx, workspaceID, operation, actorID)
}

// resolveFolder разрешает identity папки в её внутренний идентификатор.
func (s *Coordinator) resolveFolder(ctx context.Context, scope entities.WorkspaceAccess, kind string, input map[string]any) (*string, error) {
	identity := stringField(input, "folderIdentity")
	if kind == "folders" {
		entityType := entities.FolderEntityType(stringField(input, "entityType"))
		if entityType == "" {
			return nil, domainerrors.InvalidInput("folder_entity_type_required", "entityType is required")
		}
		parent := stringField(input, "parentIdentity")
		if parent == "" && !boolField(input, "isRoot") {
			parent = entities.RootFolderIdentity(entityType)
		}
		parent = resolvableFolderIdentity(parent, entityType)
		return s.repository.ResolveFolder(ctx, scope.Workspace.ID, parent, entityType)
	}
	if identity == "" {
		identity = entities.RootFolderIdentity(kind)
	}
	entityType := entities.FolderEntityType(kind)
	identity = resolvableFolderIdentity(identity, entityType)
	return s.repository.ResolveFolder(ctx, scope.Workspace.ID, identity, entityType)
}

// resolveDocumentFolder разрешает папку, указанную в документе.
func (s *Coordinator) resolveDocumentFolder(ctx context.Context, scope entities.WorkspaceAccess, doc entities.Document) (*string, error) {
	var data map[string]any
	_ = json.Unmarshal(doc.Data, &data)
	if doc.Type == "folders" {
		entityType := entities.FolderEntityType(stringField(data, "entityType"))
		parent := stringField(data, "parentIdentity")
		if parent == "" && !boolValue(data["isRoot"]) {
			parent = entities.RootFolderIdentity(entityType)
		}
		parent = resolvableFolderIdentity(parent, entityType)
		return s.repository.ResolveFolder(ctx, scope.Workspace.ID, parent, entityType)
	}
	identity := ""
	if doc.FolderIdentity != nil {
		identity = *doc.FolderIdentity
	}
	if identity == "" {
		identity = entities.RootFolderIdentity(doc.Type)
	}
	entityType := entities.FolderEntityType(doc.Type)
	identity = resolvableFolderIdentity(identity, entityType)
	return s.repository.ResolveFolder(ctx, scope.Workspace.ID, identity, entityType)
}

// resolvableFolderIdentity сопоставляет legacy-корень streams с общим корнем queries.
// Старые revisions хранят root-streams, а актуальная схема использует root-queries.
func resolvableFolderIdentity(identity, entityType string) string {
	if identity == "root-streams" && entityType == entities.FolderEntityType("streams") {
		return entities.RootFolderIdentity("streams")
	}
	return identity
}

// replaceStructuredRelations обновляет структурированные связи документа.
func (s *Coordinator) replaceStructuredRelations(ctx context.Context, document entities.Document) error {
	if document.Type != "projects" {
		return nil
	}
	var data map[string]any
	if err := json.Unmarshal(document.Data, &data); err != nil {
		return domainerrors.InvalidInput("document_data_invalid", "Document data is invalid")
	}
	environments := relationIdentityList(data["allowedEnvironments"])
	if len(environments) == 0 {
		environments = relationIdentityList(data["allowedEnvironmentIdentities"])
	}
	if err := s.repository.ReplaceProjectEnvironments(ctx, document, environments); err != nil {
		if strings.Contains(err.Error(), "relation target") {
			return domainerrors.InvalidInput("relation_target_not_found", err.Error())
		}
		return err
	}
	return nil
}

// relationIdentityList извлекает список identity связанных документов.
func relationIdentityList(value any) []string {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	seen := map[string]bool{}
	result := []string{}
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

// documentFromInput создаёт доменный документ из входных данных.
func documentFromInput(kind, workspace string, input map[string]any, actorID string) entities.Document {
	data := copyMap(input)
	for _, key := range append(readOnlyFields, "identity", "displayName", "description", "folderIdentity", "managedBy", "managedById", "meta", "active") {
		delete(data, key)
	}
	description := optionalString(input, "description")
	folder := optionalString(input, "folderIdentity")
	managedID := optionalString(input, "managedById")
	return entities.Document{ID: uuid.NewString(), WorkspaceID: workspace, Type: kind, Identity: stringField(input, "identity"), DisplayName: stringField(input, "displayName"), Description: description, FolderIdentity: folder, ManagedBy: defaultString(stringField(input, "managedBy"), "user"), ManagedByID: managedID, Meta: jsonField(input, "meta", json.RawMessage(`{}`)), Data: mustJSON(data), Active: defaultBool(input, "active", true), Revision: 1, CreatedBy: entities.Actor{ID: actorID}, UpdatedBy: entities.Actor{ID: actorID}}
}

// replaceDocumentFromInput строит полное новое состояние существующего документа.
// Локальная идентичность и audit создания сохраняются, portable-содержимое заменяется целиком.
func replaceDocumentFromInput(existing entities.Document, input map[string]any, actorID string) entities.Document {
	next := documentFromInput(existing.Type, existing.WorkspaceID, input, actorID)
	next.ID = existing.ID
	next.Revision = existing.Revision
	next.CreatedBy = existing.CreatedBy
	next.CreatedAt = existing.CreatedAt
	next.DeletedAt = nil
	return next
}

// documentAsInput преобразует доменный документ в данные для повторной проверки.
func documentAsInput(doc entities.Document) map[string]any {
	var result map[string]any
	_ = json.Unmarshal(doc.Data, &result)
	result["identity"] = doc.Identity
	result["displayName"] = doc.DisplayName
	result["managedBy"] = doc.ManagedBy
	return result
}

// commitChanges формирует изменения коммита по выполненным операциям.
func commitChanges(revisions []entities.Revision) []entities.CommitChange {
	groups := map[string][]entities.Revision{}
	order := []string{}
	for _, v := range revisions {
		key := v.DocumentType + ":" + v.DocumentID
		if _, ok := groups[key]; !ok {
			order = append(order, key)
		}
		groups[key] = append(groups[key], v)
	}
	result := []entities.CommitChange{}
	for _, key := range order {
		items := groups[key]
		first, last := items[0], items[len(items)-1]
		result = append(result, entities.CommitChange{DocumentType: last.DocumentType, DocumentID: last.DocumentID, BeforeRevisionID: first.ParentRevisionID, AfterRevisionID: &last.ID, Operation: last.Operation})
	}
	return result
}

// validateDocument проверяет общие и зависящие от типа ограничения документа.
func validateDocument(kind string, input map[string]any) error {
	if err := validateIdentity(stringField(input, "identity")); err != nil {
		return err
	}
	if stringField(input, "displayName") == "" {
		return domainerrors.InvalidInput("display_name_required", "displayName is required")
	}
	managed := defaultString(stringField(input, "managedBy"), "user")
	if !slices.Contains([]string{"user", "system", "integration"}, managed) {
		return domainerrors.InvalidInput("managed_by_invalid", "managedBy is invalid")
	}
	if source, ok := input["source"]; ok {
		value, ok := source.(string)
		if !ok {
			return domainerrors.InvalidInput("source_invalid", "source must be a string")
		}
		if len(value) > 8*1024*1024 {
			return domainerrors.InvalidInput("source_too_large", "source exceeds 8 MiB")
		}
	}
	versionedSourceKinds := []string{"types", "queries", "data-views", "stores", "streams", "updates", "filters", "computations", "compositions", "styles"}
	if slices.Contains(versionedSourceKinds, kind) {
		_, hasSource := input["source"]
		version, hasVersion := numberField(input, "sourceVersion")
		if kind == "filters" && !hasSource && !hasVersion {
			// Filter допускает декларативные fields без source.
		} else if !hasSource || !hasVersion || version <= 0 {
			return domainerrors.InvalidInput("source_contract_invalid", "source and positive sourceVersion are required")
		}
	}
	if kind == "queries" {
		version, _ := numberField(input, "sourceVersion")
		if version != 2 {
			return domainerrors.InvalidInput("query_source_version_invalid", "Query sourceVersion must be 2")
		}
	}
	if kind == "tenants" && stringField(input, "code") == "" {
		return domainerrors.InvalidInput("tenant_code_required", "code is required")
	}
	if kind == "updates" && stringField(input, "storeIdentity") == "" {
		return domainerrors.InvalidInput("update_store_required", "storeIdentity is required")
	}
	if kind == "components" {
		if _, ok := input["source"].(string); !ok {
			return domainerrors.InvalidInput("component_source_required", "source is required")
		}
		if version, ok := numberField(input, "modelVersion"); !ok || version <= 0 {
			return domainerrors.InvalidInput("component_model_version_invalid", "modelVersion must be positive")
		}
	}
	if kind == "computations" {
		if version, ok := numberField(input, "contractVersion"); !ok || version <= 0 {
			return domainerrors.InvalidInput("computation_contract_version_invalid", "contractVersion must be positive")
		}
	}
	if kind == "auth-profiles" {
		if stringField(input, "adapterId") == "" {
			return domainerrors.InvalidInput("auth_profile_adapter_required", "adapterId is required")
		}
		if !slices.Contains([]string{"localStorage", "sessionStorage", "memory"}, stringField(input, "persist")) {
			return domainerrors.InvalidInput("auth_profile_persist_invalid", "persist must be localStorage, sessionStorage or memory")
		}
		if references, exists := input["credentialRefs"]; exists {
			if err := validateCredentialRefs(references, "credentialRefs"); err != nil {
				return err
			}
		}
	}
	if kind == "vocabs" {
		if mode := stringField(input, "mode"); mode != "" && !slices.Contains([]string{"external_payload", "internal"}, mode) {
			return domainerrors.InvalidInput("vocab_mode_invalid", "mode is invalid")
		}
		if mode := stringField(input, "authMode"); mode != "" && !slices.Contains([]string{"inherit", "profile", "none"}, mode) {
			return domainerrors.InvalidInput("vocab_auth_mode_invalid", "authMode is invalid")
		}
	}
	if err := validateProjectContract(kind, input); err != nil {
		return err
	}
	if kind == "folders" {
		entityType := stringField(input, "entityType")
		if !slices.Contains(Collections, entityType) || entityType == "folders" {
			return domainerrors.InvalidInput("folder_entity_type_invalid", "entityType must be a folderable collection")
		}
		if _, exists := input["isSystem"]; exists {
			return domainerrors.InvalidInput("folder_is_system_unsupported", "isSystem is replaced by managedBy")
		}
		if boolField(input, "isRoot") {
			return domainerrors.InvalidInput("folder_root_field_read_only", "isRoot is server-managed")
		}
	}
	return nil
}

// validateProjectContract проверяет контракт проекта и запрещённые устаревшие поля.
func validateProjectContract(kind string, input map[string]any) error {
	if kind != "projects" {
		return nil
	}
	for _, field := range []string{"navigation", "navigationId", "sortOrder", "sort_order"} {
		if _, exists := input[field]; exists {
			return domainerrors.InvalidInput("project_legacy_field", "Project must use order and navigationIdentity")
		}
	}
	return nil
}

// validateIdentity проверяет обязательность и допустимую длину identity.
func validateIdentity(value string) error {
	length := len(strings.TrimSpace(value))
	if length == 0 {
		return domainerrors.InvalidInput("identity_required", "identity is required")
	}
	if length > 160 {
		return domainerrors.InvalidInput("identity_too_long", "identity must not exceed 160 characters")
	}
	return nil
}

// validateSecrets проверяет, что входные данные не содержат открытых секретов.
func validateSecrets(value any) error {
	var walk func(any, string) error
	walk = func(current any, path string) error {
		switch typed := current.(type) {
		case map[string]any:
			for key, item := range typed {
				normalized := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(key, "_", ""), "-", ""))
				if normalized == "credentialrefs" {
					if err := validateCredentialRefs(item, joinPath(path, key)); err != nil {
						return err
					}
					continue
				}
				if normalized == "manualtoken" || strings.Contains(normalized, "password") || strings.Contains(normalized, "clientsecret") || strings.Contains(normalized, "accesstoken") || strings.Contains(normalized, "refreshtoken") || normalized == "bearertoken" || normalized == "secret" {
					return domainerrors.WithDetails(domainerrors.InvalidInput("secret_field_forbidden", "Secret material must be provided through credentialRefs"), map[string]any{"field": joinPath(path, key)})
				}
				if err := walk(item, joinPath(path, key)); err != nil {
					return err
				}
			}
		case []any:
			for index, item := range typed {
				if err := walk(item, fmt.Sprintf("%s[%d]", path, index)); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return walk(value, "")
}

// validateCredentialRefs проверяет корректность ссылок на учётные данные.
func validateCredentialRefs(value any, path string) error {
	switch references := value.(type) {
	case map[string]any:
		for name, value := range references {
			reference, ok := value.(string)
			if strings.TrimSpace(name) == "" || !ok || strings.TrimSpace(reference) == "" {
				return domainerrors.WithDetails(domainerrors.InvalidInput("credential_ref_invalid", "Each credential reference must be a non-empty string"), map[string]any{"field": joinPath(path, name)})
			}
		}
	case []any:
		for index, value := range references {
			reference, ok := value.(string)
			if !ok || strings.TrimSpace(reference) == "" {
				return domainerrors.WithDetails(domainerrors.InvalidInput("credential_ref_invalid", "Each credential reference must be a non-empty string"), map[string]any{"field": fmt.Sprintf("%s[%d]", path, index)})
			}
		}
	default:
		return domainerrors.WithDetails(domainerrors.InvalidInput("credential_refs_invalid", "credentialRefs must contain named external references"), map[string]any{"field": path})
	}
	return nil
}

// rejectReadOnly отклоняет изменение серверных полей только для чтения.
func rejectReadOnly(input map[string]any) error {
	for _, field := range readOnlyFields {
		if _, ok := input[field]; ok {
			return domainerrors.WithDetails(domainerrors.InvalidInput("read_only_field", "Actor and audit fields are read-only"), map[string]any{"field": field})
		}
	}
	return nil
}

// checksumContent вычисляет контрольную сумму содержимого документа.
func checksumContent(doc entities.Document) string {
	return checksum(mustJSON(map[string]any{"identity": doc.Identity, "displayName": doc.DisplayName, "description": doc.Description, "folderIdentity": doc.FolderIdentity, "managedBy": doc.ManagedBy, "managedById": doc.ManagedByID, "meta": doc.Meta, "data": doc.Data, "active": doc.Active, "deletedAt": doc.DeletedAt}))
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
		workspace.Description = optionalString(patch, "description")
	}
	if value, ok := patch["dataMode"].(string); ok {
		workspace.DataMode = value
	}
	if value, ok := patch["configuration"]; ok {
		workspace.Configuration = mustJSON(value)
	}
	if value, ok := patch["meta"]; ok {
		workspace.Meta = mustJSON(value)
	}
	if value, ok := patch["active"].(bool); ok {
		workspace.Active = value
	}
	return workspace
}

// checksumWorkspace вычисляет контрольную сумму рабочего пространства.
func checksumWorkspace(workspace entities.Workspace) string {
	return checksum(mustJSON(map[string]any{"identity": workspace.Identity, "displayName": workspace.DisplayName, "description": workspace.Description, "dataMode": workspace.DataMode, "configuration": workspace.Configuration, "meta": workspace.Meta, "active": workspace.Active}))
}

// checksumIntegration вычисляет контрольную сумму интеграции.
func checksumIntegration(integration entities.Integration) string {
	return checksum(mustJSON(map[string]any{"identity": integration.Identity, "displayName": integration.DisplayName, "description": integration.Description, "version": integration.Version, "managedBy": integration.ManagedBy, "managedById": integration.ManagedByID, "meta": integration.Meta, "active": integration.Active, "deletedAt": integration.DeletedAt}))
}

// checksum вычисляет SHA-256 контрольную сумму сериализованных данных.
func checksum(raw []byte) string { sum := sha256.Sum256(raw); return hex.EncodeToString(sum[:]) }

// mustJSON сериализует значение в JSON для внутреннего конвейера.
func mustJSON(value any) json.RawMessage { raw, _ := json.Marshal(value); return raw }

// jsonField извлекает поле и сериализует его в JSON.
func jsonField(input map[string]any, key string, fallback json.RawMessage) json.RawMessage {
	value, ok := input[key]
	if !ok {
		return fallback
	}
	return mustJSON(value)
}

// stringField извлекает и нормализует строковое поле.
func stringField(input map[string]any, key string) string {
	value, _ := input[key].(string)
	return strings.TrimSpace(value)
}

// optionalString извлекает необязательное строковое поле.
func optionalString(input map[string]any, key string) *string {
	value, ok := input[key]
	if !ok || value == nil {
		return nil
	}
	text, ok := value.(string)
	if !ok {
		return nil
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	return &text
}

// boolField извлекает логическое поле.
func boolField(input map[string]any, key string) bool { v, _ := input[key].(bool); return v }

// boolValue извлекает логическое значение с признаком наличия.
func boolValue(value any) bool { v, _ := value.(bool); return v }

// defaultBool возвращает логическое поле или значение по умолчанию.
func defaultBool(input map[string]any, key string, fallback bool) bool {
	v, ok := input[key].(bool)
	if !ok {
		return fallback
	}
	return v
}

// numberField извлекает целочисленное поле без потери точности.
func numberField(input map[string]any, key string) (int, bool) {
	switch value := input[key].(type) {
	case float64:
		return int(value), value == float64(int(value))
	case int:
		return value, true
	case json.Number:
		v, e := value.Int64()
		return int(v), e == nil
	default:
		return 0, false
	}
}

// defaultString возвращает строку или значение по умолчанию.
func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

// copyMap создаёт поверхностную копию карты данных.
func copyMap(input map[string]any) map[string]any {
	result := map[string]any{}
	for key, value := range input {
		result[key] = value
	}
	return result
}

// joinPath безопасно объединяет части пути переносимого пакета.
func joinPath(left, right string) string {
	if left == "" {
		return right
	}
	return left + "." + right
}

// integrationItems преобразует интеграции в элементы переносимого пакета.
func integrationItems(input map[string]any) ([]map[string]any, error) {
	value, ok := input["installedIntegrations"]
	if !ok || value == nil {
		return nil, nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, domainerrors.InvalidInput("installed_integrations_invalid", "installedIntegrations is invalid")
	}
	var result []map[string]any
	if err = json.Unmarshal(raw, &result); err != nil {
		return nil, domainerrors.InvalidInput("installed_integrations_invalid", "installedIntegrations must be an array")
	}
	for _, item := range result {
		if _, ok := item["configuration"]; !ok {
			item["configuration"] = map[string]any{}
		}
	}
	return result, nil
}

type portableBundleNormalization struct {
	IgnoredIntegrations        int
	IgnoredDeletedDocuments    int
	IgnoredLegacyFolders       int
	NormalizedFolderReferences int
}

// normalizePortableBundle переводит legacy portable-артефакты в контракт backend import.
func normalizePortableBundle(bundle *entities.PortableBundle) portableBundleNormalization {
	result := portableBundleNormalization{}
	if bundle == nil {
		return result
	}
	if bundle.Documents == nil {
		bundle.Documents = map[string][]map[string]any{}
	}
	if bundle.Kind == "" {
		bundle.Kind = "workspace-snapshot"
	}
	delete(bundle.Workspace, "state")
	result.IgnoredIntegrations = len(bundle.InstalledIntegrations)
	bundle.InstalledIntegrations = []map[string]any{}
	if legacy, ok := bundle.Documents["componentSFCs"]; ok {
		bundle.Documents["components"] = append(bundle.Documents["components"], legacy...)
		delete(bundle.Documents, "componentSFCs")
	}
	for _, folder := range bundle.Documents["folders"] {
		if entityType := stringField(folder, "entityType"); entityType != "" {
			folder["entityType"] = entities.FolderEntityType(entityType)
		}
		if parent := stringField(folder, "parentIdentity"); parent == "root-streams" {
			folder["parentIdentity"] = entities.RootFolderIdentity("streams")
		}
	}

	ignoredDeletedFolders := map[string]bool{"soft-deleted": true}
	folderTypes := map[string]string{}
	for _, folder := range bundle.Documents["folders"] {
		folderTypes[stringField(folder, "identity")] = stringField(folder, "entityType")
	}
	for changed := true; changed; {
		changed = false
		for _, folder := range bundle.Documents["folders"] {
			identity := stringField(folder, "identity")
			if identity != "" && ignoredDeletedFolders[stringField(folder, "parentIdentity")] && !ignoredDeletedFolders[identity] {
				ignoredDeletedFolders[identity] = true
				changed = true
			}
		}
	}

	for kind, items := range bundle.Documents {
		filtered := make([]map[string]any, 0, len(items))
		for _, item := range items {
			delete(item, "state")
			if kind == "queries" {
				if sourceVersion, ok := numberField(item, "sourceVersion"); ok && sourceVersion == 1 {
					item["sourceVersion"] = 2
				}
			}
			if kind == "folders" {
				identity := stringField(item, "identity")
				if ignoredDeletedFolders[identity] || identity == "no-folder" || identity == "root-bindings" || identity == "root-streams" {
					result.IgnoredLegacyFolders++
					continue
				}
			}
			if kind != "folders" && ignoredDeletedFolders[stringField(item, "folderIdentity")] {
				result.IgnoredDeletedDocuments++
				continue
			}
			if kind != "folders" {
				folderIdentity := stringField(item, "folderIdentity")
				if kind == "streams" && folderIdentity == "root-streams" {
					folderIdentity = entities.RootFolderIdentity(kind)
				}
				if folderIdentity != stringField(item, "folderIdentity") {
					item["folderIdentity"] = folderIdentity
					result.NormalizedFolderReferences++
				}
				switch folderIdentity {
				case "no-folder", "root-bindings":
					delete(item, "folderIdentity")
					result.NormalizedFolderReferences++
				default:
					folderType, hasFolderType := folderTypes[folderIdentity]
					expectedFolderType := entities.FolderEntityType(kind)
					if folderIdentity != "" && ((strings.HasPrefix(folderIdentity, "root-") && folderIdentity != entities.RootFolderIdentity(kind)) || (hasFolderType && folderType != expectedFolderType)) {
						item["folderIdentity"] = entities.RootFolderIdentity(kind)
						result.NormalizedFolderReferences++
					}
				}
			}
			filtered = append(filtered, item)
		}
		bundle.Documents[kind] = filtered
	}
	return result
}

// orderPortableItems упорядочивает элементы пакета с учётом зависимостей.
func orderPortableItems(kind string, items []map[string]any) ([]map[string]any, error) {
	if kind != "folders" || len(items) < 2 {
		return items, nil
	}
	remaining := map[string]map[string]any{}
	for _, item := range items {
		identity := stringField(item, "identity")
		if identity == "" {
			return nil, domainerrors.InvalidInput("identity_required", "Folder identity is required")
		}
		remaining[identity] = item
	}
	result := make([]map[string]any, 0, len(items))
	for len(remaining) > 0 {
		identities := make([]string, 0, len(remaining))
		for identity := range remaining {
			identities = append(identities, identity)
		}
		slices.Sort(identities)
		progress := false
		for _, identity := range identities {
			item := remaining[identity]
			parent := stringField(item, "parentIdentity")
			if parent != "" {
				if _, waitsForParent := remaining[parent]; waitsForParent {
					continue
				}
			}
			result = append(result, item)
			delete(remaining, identity)
			progress = true
		}
		if !progress {
			return nil, domainerrors.InvalidInput("folder_cycle", "Portable bundle contains a folder cycle")
		}
	}
	return result, nil
}

// applyStructuredIdentityMap переписывает структурированные ссылки по карте identity.
func applyStructuredIdentityMap(kind string, item map[string]any, identityMap map[string]string) {
	mapField := func(field, targetType string) {
		value := stringField(item, field)
		if mapped, ok := identityMap[targetType+":"+value]; value != "" && ok {
			item[field] = mapped
		}
	}
	mapField("folderIdentity", "folders")
	switch kind {
	case "folders":
		mapField("parentIdentity", "folders")
	case "updates":
		mapField("storeIdentity", "stores")
	case "vocabs":
		mapField("authProfileIdentity", "auth-profiles")
	case "projects":
		mapField("navigationIdentity", "navigations")
		for _, field := range []string{"allowedEnvironments", "allowedEnvironmentIdentities"} {
			values, ok := item[field].([]any)
			if !ok {
				continue
			}
			for index, value := range values {
				identity, ok := value.(string)
				if !ok {
					continue
				}
				if mapped, exists := identityMap["environments:"+identity]; exists {
					values[index] = mapped
				}
			}
		}
	}
}

// preconditionError создаёт ошибку отсутствующей обязательной предусловной версии.
func preconditionError() error {
	return domainerrors.New("precondition_required", "If-Match header is required", 428)
}

// revisionConflict создаёт ошибку конфликта ревизий.
func revisionConflict() error {
	return domainerrors.Conflict("revision_conflict", "Document revision does not match If-Match")
}

// unsupported создаёт ошибку неподдерживаемой операции восстановления.
func unsupported(kind string) error {
	return domainerrors.WithDetails(domainerrors.InvalidInput("collection_unsupported", "Collection is not supported by this MVP"), map[string]any{"collection": kind})
}

// mapNotFound преобразует ошибку отсутствующей записи в доменную ошибку.
func mapNotFound(err error) error {
	if errors.Is(err, ports.ErrNotFound) {
		return domainerrors.NotFound("not_found", "Entity not found")
	}
	return err
}

// mapConflict преобразует конфликт хранилища в доменную ошибку.
func mapConflict(err error) error {
	if err == nil {
		return nil
	}
	text := strings.ToLower(err.Error())
	if strings.Contains(text, "revision conflict") {
		return revisionConflict()
	}
	if strings.Contains(text, "duplicate") || strings.Contains(text, "unique constraint") || strings.Contains(text, "23505") {
		return domainerrors.Conflict("identity_conflict", "Identity already exists")
	}
	if strings.Contains(text, "relation target") {
		return domainerrors.InvalidInput("relation_target_not_found", err.Error())
	}
	return err
}
