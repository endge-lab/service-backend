package configuration

// EnsureWorkspaceDefaults инициализирует конфигурацию нового workspace.
// Явно переданные поля сохраняются; defaults не применяются к overrides документов.
func EnsureWorkspaceDefaults(value any) any {
	configuration, ok := configurationObject(value)
	if !ok {
		return value
	}
	defaults := map[string]any{
		"vars": []any{},
		"locales": []any{
			map[string]any{"code": "ru", "displayName": "Русский", "shortLabel": "RU", "direction": "ltr"},
			map[string]any{"code": "en", "displayName": "English", "shortLabel": "EN", "direction": "ltr"},
		},
		"defaultLocale":  "en",
		"fallbackLocale": "en",
		"themes": []any{
			map[string]any{"identity": "light", "displayName": "Светлая"},
			map[string]any{"identity": "dark", "displayName": "Тёмная"},
		},
		"defaultTheme": "dark",
		"timezones": []any{
			map[string]any{"identity": "local", "displayName": "Локальное время"},
			map[string]any{"identity": "UTC", "displayName": "UTC"},
		},
		"defaultTimezone":            "local",
		"defaultAuthProfileIdentity": nil,
		"sfcAdapterIds":              []any{"vue-native"},
		"defaultSfcAdapterId":        "vue-native",
		"diagnostics":                map[string]any{},
		"values":                     map[string]any{},
	}
	for key, defaultValue := range defaults {
		if _, exists := configuration[key]; !exists {
			configuration[key] = defaultValue
		}
	}
	return EnsureSFCEditingDefaults(configuration)
}
