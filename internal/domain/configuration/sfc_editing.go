package configuration

import "encoding/json"

// EnsureSFCEditingDefaults добавляет persisted defaults, не заменяя существующие настройки.
func EnsureSFCEditingDefaults(value any) any {
	configuration, ok := configurationObject(value)
	if !ok {
		return value
	}
	defaults := defaultSFCEditingConfiguration()
	editing, editingOK := configuration["sfcEditing"].(map[string]any)
	if !editingOK {
		configuration["sfcEditing"] = defaults
		return configuration
	}
	for key, defaultValue := range defaults {
		if _, exists := editing[key]; !exists {
			editing[key] = defaultValue
		}
	}
	return configuration
}

func configurationObject(value any) (map[string]any, bool) {
	if value == nil {
		return map[string]any{}, true
	}
	if configuration, ok := value.(map[string]any); ok {
		return configuration, true
	}
	var configuration map[string]any
	switch raw := value.(type) {
	case json.RawMessage:
		if err := json.Unmarshal(raw, &configuration); err == nil && configuration != nil {
			return configuration, true
		}
	case []byte:
		if err := json.Unmarshal(raw, &configuration); err == nil && configuration != nil {
			return configuration, true
		}
	}
	return nil, false
}

func defaultSFCEditingConfiguration() map[string]any {
	return map[string]any{
		"cancelOn": []any{
			map[string]any{"event": "keydown", "key": []any{"Escape"}, "prevent": true, "stop": true},
			map[string]any{"event": "focusout"},
		},
		"commitOn": []any{
			map[string]any{"event": "keydown", "key": []any{"Enter"}, "prevent": true},
		},
	}
}
