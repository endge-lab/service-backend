package configuration

import "encoding/json"

var configurationDocumentTypes = map[string]struct{}{
	"environments": {},
	"projects":     {},
	"tenants":      {},
}

// RemoveLegacySSE удаляет глобальную SSE-настройку из полной workspace-конфигурации.
func RemoveLegacySSE(value any) {
	configuration, ok := value.(map[string]any)
	if !ok {
		return
	}
	delete(configuration, "sse")
}

// RemoveLegacySSEFromDocument удаляет SSE override из configuration contribution документа.
func RemoveLegacySSEFromDocument(documentType string, values map[string]any) {
	if _, ok := configurationDocumentTypes[documentType]; !ok {
		return
	}
	configuration, ok := values["configuration"].(map[string]any)
	if !ok {
		return
	}
	delete(configuration, "sse")
	for _, key := range []string{"patch", "value"} {
		if nested, ok := configuration[key].(map[string]any); ok {
			delete(nested, "sse")
		}
	}
}

// RemoveLegacySSEFromDocumentData очищает JSON data конфигурационного документа.
func RemoveLegacySSEFromDocumentData(documentType string, raw json.RawMessage) json.RawMessage {
	var values map[string]any
	if err := json.Unmarshal(raw, &values); err != nil || values == nil {
		return raw
	}
	RemoveLegacySSEFromDocument(documentType, values)
	result, err := json.Marshal(values)
	if err != nil {
		return raw
	}
	return result
}
