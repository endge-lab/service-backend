package documents

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"slices"
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
	"github.com/google/uuid"
)

var Collections = []string{"projects", "tenants", "environments", "folders", "types", "queries", "data-views", "compositions", "stores", "streams", "updates", "mocks", "components", "actions", "filters", "converters", "computations", "vocabs", "i18n-bundles", "auth-profiles", "navigations", "styles"}

var sourceVersionCollections = []string{"types", "queries", "data-views", "compositions", "stores", "streams", "updates", "filters", "computations", "styles"}

var readOnlyFields = []string{"id", "type", "revision", "author", "createdBy", "updatedBy", "createdAt", "updatedAt", "deletedAt", "created_by", "updated_by"}

// validateCollection проверяет, что коллекция поддерживается документным API.
func validateCollection(collection string) error {
	if slices.Contains(Collections, collection) {
		return nil
	}
	return domainerrors.WithDetails(domainerrors.InvalidInput("collection_unsupported", "Collection is not supported by this MVP"), map[string]any{"collection": collection})
}

// validateDocument проверяет общие и зависящие от типа ограничения документа.
func validateDocument(kind string, input map[string]any) error {
	if err := validateIdentity(stringField(input, "identity")); err != nil {
		return err
	}
	if stringField(input, "displayName") == "" {
		return domainerrors.InvalidInput("display_name_required", "displayName is required")
	}
	managedBy := defaultString(stringField(input, "managedBy"), "user")
	if !slices.Contains([]string{"user", "system", "integration"}, managedBy) {
		return domainerrors.InvalidInput("managed_by_invalid", "managedBy is invalid")
	}
	if source, ok := input["source"]; ok {
		value, valid := source.(string)
		if !valid {
			return domainerrors.InvalidInput("source_invalid", "source must be a string")
		}
		if len(value) > 8*1024*1024 {
			return domainerrors.InvalidInput("source_too_large", "source exceeds 8 MiB")
		}
		if slices.Contains(sourceVersionCollections, kind) {
			version, valid := numberField(input, "sourceVersion")
			if !valid || version <= 0 {
				return domainerrors.InvalidInput("source_version_invalid", "sourceVersion must be positive")
			}
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
	if kind == "projects" {
		if _, exists := input["navigation"]; exists {
			return domainerrors.InvalidInput("project_navigation_legacy", "Project navigation must use navigationIdentity")
		}
		if _, exists := input["navigationId"]; exists {
			return domainerrors.InvalidInput("project_navigation_legacy", "Project navigation must use navigationIdentity")
		}
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
	if kind == "auth-profiles" {
		publicFields := copyMap(input)
		delete(publicFields, "config")
		return validateSecrets(publicFields)
	}
	return validateSecrets(input)
}

// rejectReadOnly отклоняет изменение серверных полей только для чтения.
func rejectReadOnly(input map[string]any) error {
	return shared.RejectReadOnly(input)
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
	return shared.ValidateSecrets(value)
}

// documentFromInput создаёт доменный документ из входных данных.
func documentFromInput(kind, workspaceID string, input map[string]any, actorID string) entities.Document {
	data := copyMap(input)
	for _, key := range append(readOnlyFields, "identity", "displayName", "description", "folderIdentity", "managedBy", "managedById", "meta", "active") {
		delete(data, key)
	}
	return entities.Document{
		ID: uuid.NewString(), WorkspaceID: workspaceID, Type: kind,
		Identity: stringField(input, "identity"), DisplayName: stringField(input, "displayName"),
		Description: optionalString(input, "description"), FolderIdentity: optionalString(input, "folderIdentity"),
		ManagedBy: defaultString(stringField(input, "managedBy"), "user"), ManagedByID: optionalString(input, "managedById"),
		Meta: jsonField(input, "meta", json.RawMessage(`{}`)), Data: mustJSON(data), Active: defaultBool(input, "active", true),
		Revision: 1, CreatedBy: entities.Actor{ID: actorID}, UpdatedBy: entities.Actor{ID: actorID},
	}
}

// applyPatch применяет частичное обновление к документу.
func applyPatch(document entities.Document, patch map[string]any, actorID string) entities.Document {
	if value, ok := patch["identity"].(string); ok {
		document.Identity = strings.TrimSpace(value)
	}
	if value, ok := patch["displayName"].(string); ok {
		document.DisplayName = value
	}
	if _, ok := patch["description"]; ok {
		document.Description = optionalString(patch, "description")
	}
	if _, ok := patch["folderIdentity"]; ok {
		document.FolderIdentity = optionalString(patch, "folderIdentity")
	}
	if value, ok := patch["managedBy"].(string); ok {
		document.ManagedBy = value
	}
	if _, ok := patch["managedById"]; ok {
		document.ManagedByID = optionalString(patch, "managedById")
	}
	if value, ok := patch["meta"]; ok {
		document.Meta = mustJSON(value)
	}
	if value, ok := patch["active"].(bool); ok {
		document.Active = value
	}
	var data map[string]any
	_ = json.Unmarshal(document.Data, &data)
	for key, value := range patch {
		if !slices.Contains(append(readOnlyFields, "identity", "displayName", "description", "folderIdentity", "managedBy", "managedById", "meta", "active"), key) {
			data[key] = value
		}
	}
	document.Data, document.UpdatedBy = mustJSON(data), entities.Actor{ID: actorID}
	return document
}

// documentAsInput преобразует доменный документ в данные для повторной проверки.
func documentAsInput(document entities.Document) map[string]any {
	result := map[string]any{}
	_ = json.Unmarshal(document.Data, &result)
	result["identity"], result["displayName"], result["managedBy"] = document.Identity, document.DisplayName, document.ManagedBy
	return result
}

// checksumContent вычисляет контрольную сумму содержимого документа.
func checksumContent(document entities.Document) string {
	return checksum(mustJSON(map[string]any{"identity": document.Identity, "displayName": document.DisplayName, "description": document.Description, "folderIdentity": document.FolderIdentity, "managedBy": document.ManagedBy, "managedById": document.ManagedByID, "meta": canonicalJSONValue(document.Meta), "data": canonicalJSONValue(document.Data), "active": document.Active, "deletedAt": document.DeletedAt}))
}

// canonicalJSONValue устраняет различия форматирования JSONB перед no-op сравнением.
func canonicalJSONValue(raw json.RawMessage) any {
	var value any
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil
	}
	return value
}

// checksum вычисляет SHA-256 контрольную сумму сериализованных данных.
func checksum(raw []byte) string { sum := sha256.Sum256(raw); return hex.EncodeToString(sum[:]) }

// mustJSON сериализует значение в JSON для внутреннего конвейера.
func mustJSON(value any) json.RawMessage { raw, _ := json.Marshal(value); return raw }

// stringField извлекает и нормализует строковое поле.
func stringField(input map[string]any, key string) string {
	value, _ := input[key].(string)
	return strings.TrimSpace(value)
}

// optionalString извлекает необязательное строковое поле.
func optionalString(input map[string]any, key string) *string {
	value, ok := input[key].(string)
	if !ok || strings.TrimSpace(value) == "" {
		return nil
	}
	value = strings.TrimSpace(value)
	return &value
}

// boolField извлекает логическое поле.
func boolField(input map[string]any, key string) bool { value, _ := input[key].(bool); return value }

// defaultBool возвращает логическое поле или значение по умолчанию.
func defaultBool(input map[string]any, key string, fallback bool) bool {
	value, ok := input[key].(bool)
	if !ok {
		return fallback
	}
	return value
}

// defaultString возвращает строку или значение по умолчанию.
func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

// jsonField извлекает поле и сериализует его в JSON.
func jsonField(input map[string]any, key string, fallback json.RawMessage) json.RawMessage {
	value, ok := input[key]
	if !ok {
		return fallback
	}
	return mustJSON(value)
}

// copyMap создаёт поверхностную копию карты данных.
func copyMap(input map[string]any) map[string]any {
	result := map[string]any{}
	for key, value := range input {
		result[key] = value
	}
	return result
}

// numberField извлекает целочисленное поле без потери точности.
func numberField(input map[string]any, key string) (int, bool) {
	switch value := input[key].(type) {
	case float64:
		return int(value), value == float64(int(value))
	case int:
		return value, true
	case json.Number:
		number, err := value.Int64()
		return int(number), err == nil
	default:
		return 0, false
	}
}
