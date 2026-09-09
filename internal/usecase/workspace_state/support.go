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

	"github.com/endge-lab/service-backend/internal/domain/domainversion"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
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
	value := entities.Revision{ID: uuid.NewString(), WorkspaceID: doc.WorkspaceID, DocumentType: doc.Type, DocumentID: doc.ID, DocumentIdentity: doc.Identity, RevisionNumber: doc.Revision, WorkspaceSequence: &sequence, Operation: operation, ParentRevisionID: parent, RestoredFromRevisionID: restored, MutationBatchID: batch, SnapshotVersion: workspaceRevisionSnapshotVersion, Snapshot: snapshot, Checksum: checksum(snapshot), CreatedBy: doc.UpdatedBy}
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
	value := entities.Revision{ID: uuid.NewString(), WorkspaceID: workspace.ID, DocumentType: "workspaces", DocumentID: workspace.ID, DocumentIdentity: workspace.Identity, RevisionNumber: workspace.Revision, WorkspaceSequence: &sequence, Operation: operation, ParentRevisionID: parent, MutationBatchID: batch, SnapshotVersion: workspaceRevisionSnapshotVersion, Snapshot: snapshot, Checksum: checksum(snapshot), CreatedBy: workspace.UpdatedBy}
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
	if kind == entities.CollectionConfigurations {
		return nil, nil
	}
	identity := stringField(input, "folderIdentity")
	if kind == entities.CollectionFolders {
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
	if doc.Type == entities.CollectionConfigurations {
		return nil, nil
	}
	identity := ""
	entityType := entities.FolderEntityType(doc.Type)
	if doc.Type == entities.CollectionFolders {
		var data map[string]any
		_ = json.Unmarshal(doc.Data, &data)
		entityType = entities.FolderEntityType(stringField(data, "entityType"))
		identity = stringField(data, "parentIdentity")
		if identity == "" && !boolValue(data["isRoot"]) {
			identity = entities.RootFolderIdentity(entityType)
		}
	} else {
		if doc.FolderIdentity != nil {
			identity = *doc.FolderIdentity
		}
		if identity == "" {
			identity = entities.RootFolderIdentity(doc.Type)
		}
	}
	identity = resolvableFolderIdentity(identity, entityType)
	if doc.DeletedAt != nil && identity != "" {
		// Exact restore may already have soft-deleted the parent folder.
		// Preserve the existing folder relation of the deleted document.
		folder, err := s.repository.GetDocument(ctx, scope.Workspace.ID, entities.CollectionFolders, identity, true)
		if err != nil {
			return nil, err
		}
		var data map[string]any
		_ = json.Unmarshal(folder.Data, &data)
		if entities.FolderEntityType(stringField(data, "entityType")) != entityType {
			return nil, domainerrors.InvalidInput("folder_entity_type_invalid", "Folder entity type does not match document")
		}
		return &folder.ID, nil
	}
	return s.repository.ResolveFolder(ctx, scope.Workspace.ID, identity, entityType)
}

// resolvableFolderIdentity сопоставляет legacy-корень streams с общим корнем queries.
// Старые revisions хранят root-streams, а актуальная схема использует root-queries.
func resolvableFolderIdentity(identity, entityType string) string {
	if identity == entities.LegacyRootStreamsIdentity && entityType == entities.FolderEntityType(entities.CollectionStreams) {
		return entities.RootFolderIdentity(entities.CollectionStreams)
	}
	return identity
}

// replaceStructuredRelations обновляет структурированные связи документа.
func (s *Coordinator) replaceStructuredRelations(ctx context.Context, document entities.Document) error {
	if document.Type != entities.CollectionProjects {
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
	if !slices.Contains([]string{"user", entities.ManagedBySystem, "integration"}, managed) {
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
	versionedSourceKinds := []string{entities.CollectionTypes, entities.CollectionQueries, entities.CollectionDataViews, entities.CollectionStores, entities.CollectionStreams, entities.CollectionSimulations, entities.CollectionUpdates, entities.CollectionActions, entities.CollectionFilters, entities.CollectionComputations, entities.CollectionCompositions, entities.CollectionStyles, entities.CollectionConfigurations}
	if slices.Contains(versionedSourceKinds, kind) {
		_, hasSource := input["source"]
		version, hasVersion := numberField(input, "sourceVersion")
		if kind == "filters" && !hasSource && !hasVersion {
			// Filter допускает декларативные fields без source.
		} else if !hasSource || !hasVersion || version <= 0 {
			return domainerrors.InvalidInput("source_contract_invalid", "source and positive sourceVersion are required")
		}
	}
	if kind == entities.CollectionCompositions && strings.EqualFold(strings.TrimSpace(stringField(input, "kind")), "project") {
		return domainerrors.InvalidInput("composition_project_owner_unsupported", "Project owns its own Source; Composition kind project is not supported")
	}
	if kind == "queries" {
		version, _ := numberField(input, "sourceVersion")
		if version != 2 {
			return domainerrors.InvalidInput("query_source_version_invalid", "Query sourceVersion must be 2")
		}
	}
	if kind == entities.CollectionActions && strings.TrimSpace(stringField(input, "source")) == "" {
		return domainerrors.InvalidInput("action_source_invalid", "Action source must not be empty")
	}
	if kind == entities.CollectionProjects {
		version, hasVersion := numberField(input, "sourceVersion")
		if _, hasSource := input["source"].(string); !hasSource || !hasVersion || version != 1 {
			return domainerrors.InvalidInput("project_source_contract_invalid", "Project source and sourceVersion 1 are required")
		}
	}

	if kind == entities.CollectionSimulations {
		version, hasVersion := numberField(input, "sourceVersion")
		if _, hasSource := input["source"].(string); !hasSource || !hasVersion || version != 1 {
			return domainerrors.InvalidInput("simulation_source_version_invalid", "Simulation source and sourceVersion 1 are required")
		}
	}
	if kind == entities.CollectionConfigurations {
		version, hasVersion := numberField(input, "sourceVersion")
		if _, hasSource := input["source"].(string); !hasSource || !hasVersion || version != 1 {
			return domainerrors.InvalidInput("configuration_source_version_invalid", "Configuration source and sourceVersion 1 are required")
		}
		if stringField(input, "folderIdentity") != "" {
			return domainerrors.InvalidInput("configuration_folder_unsupported", "Configuration documents do not support folders")
		}
	}
	if kind == entities.CollectionTenants && stringField(input, "code") == "" {
		return domainerrors.InvalidInput("tenant_code_required", "code is required")
	}
	if kind == entities.CollectionUpdates && stringField(input, "storeIdentity") == "" {
		return domainerrors.InvalidInput("update_store_required", "storeIdentity is required")
	}
	if kind == entities.CollectionComponents {
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
	if kind == entities.CollectionAuthProfiles {
		if err := shared.ValidateAuthProfile(input); err != nil {
			return err
		}
	}
	if kind == entities.CollectionVocabs {
		source, hasSource := input["source"].(string)
		version, hasVersion := numberField(input, "sourceVersion")
		if hasSource != hasVersion {
			return domainerrors.InvalidInput("vocab_source_contract_invalid", "Vocab source and sourceVersion must be provided together")
		}
		if hasSource && strings.TrimSpace(source) == "" {
			return domainerrors.InvalidInput("vocab_source_invalid", "Vocab source must not be empty")
		}
		if hasVersion && version != 1 {
			return domainerrors.InvalidInput("vocab_source_version_invalid", "Vocab sourceVersion must be 1")
		}
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
	if kind == entities.CollectionFolders {
		entityType := stringField(input, "entityType")
		if !slices.Contains(Collections, entityType) || entityType == entities.CollectionFolders {
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
	if kind != entities.CollectionProjects {
		return nil
	}
	for _, field := range []string{"navigation", "navigationId", "navigationIdentity", "sortOrder", "sort_order"} {
		if _, exists := input[field]; exists {
			return domainerrors.InvalidInput("project_legacy_field", "Project must use order and cannot reference navigation")
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
func validateSecrets(value any) error { return shared.ValidateSecrets(value) }

// checksumContent вычисляет контрольную сумму содержимого документа.
func checksumContent(doc entities.Document) string {
	return checksum(mustJSON(map[string]any{"identity": doc.Identity, "displayName": doc.DisplayName, "description": doc.Description, "folderIdentity": doc.FolderIdentity, "managedBy": doc.ManagedBy, "managedById": doc.ManagedByID, "meta": doc.Meta, "data": doc.Data, "active": doc.Active, "deletedAt": doc.DeletedAt}))
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

type portableBundleNormalization struct {
	IgnoredIntegrations        int
	IgnoredLegacyFolders       int
	NormalizedFolderReferences int
	MigratedLegacyActions      int
	MigratedLegacyVocabs       int
}

// normalizePortableBundleForImport validates the source artifact according to
// its declared hash contract before storing the current canonical form.
func normalizePortableBundleForImport(bundle *entities.PortableBundle) (string, portableBundleNormalization, error) {
	if bundle == nil {
		return "", portableBundleNormalization{}, fmt.Errorf("portable bundle is required")
	}
	var computedSourceDomainVersion string
	var err error
	if bundle.DomainVersion == "" {
		computedSourceDomainVersion, err = domainversion.Compute(*bundle)
	} else {
		computedSourceDomainVersion, err = domainversion.ComputeForDeclaredVersion(*bundle, bundle.DomainVersion)
	}
	if err != nil {
		return "", portableBundleNormalization{}, err
	}
	return computedSourceDomainVersion, normalizePortableBundle(bundle), nil
}

// normalizePortableBundle applies the shared portable-domain canonicalizer and
// then removes target-local integrations according to import policy.
func normalizePortableBundle(bundle *entities.PortableBundle) portableBundleNormalization {
	if bundle == nil {
		return portableBundleNormalization{}
	}
	report := domainversion.CanonicalizeInPlace(bundle)
	result := portableBundleNormalization{
		IgnoredIntegrations:        len(bundle.InstalledIntegrations),
		IgnoredLegacyFolders:       report.IgnoredLegacyFolders,
		NormalizedFolderReferences: report.NormalizedFolderReferences,
		MigratedLegacyActions:      report.MigratedLegacyActions,
		MigratedLegacyVocabs:       report.MigratedLegacyVocabs,
	}
	bundle.InstalledIntegrations = []map[string]any{}
	return result
}

// finalizeImportDomainVersion переводит уже проверенный и нормализованный snapshot
// в актуальное persisted-представление для применения.
func finalizeImportDomainVersion(bundle *entities.PortableBundle, providedDomainVersion string) (bool, error) {
	report := domainversion.CanonicalizeInPlace(bundle)
	effectiveDomainVersion, err := domainversion.Compute(*bundle)
	if err != nil {
		return false, err
	}
	if providedDomainVersion != "" {
		bundle.DomainVersion = effectiveDomainVersion
	} else {
		bundle.DomainVersion = ""
	}
	return report.SFCEditingDefaultsAdded, nil
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
