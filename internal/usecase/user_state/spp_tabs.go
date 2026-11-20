package user_state

import (
	"encoding/json"
	"slices"
)

const maxSPPTabID int64 = 1<<53 - 1

var sppTabKinds = []string{"arrival", "departure", "groundHandling", "timeline"}

var sppTabsWorkspaceDefinition = Definition{
	Key: "aodb.spp-tabs.workspace",
	Description: map[string]string{
		"ru": "Вкладки СПП, активная вкладка и настройки фильтров. Если при открытии СПП список пуст, AODB откроет вкладки из конфигурации рабочего пространства.",
		"en": "Flight schedule tabs, the active tab, and filter settings. If the saved list is empty when the schedule opens, AODB opens the tabs defined by the workspace configuration.",
	},
	Schema: map[string]any{
		"type":     "object",
		"required": []string{"version", "tabs"},
		"properties": map[string]any{
			"version":  map[string]any{"type": "integer", "const": 1},
			"activeId": map[string]any{"type": []string{"integer", "null"}, "minimum": 1, "maximum": maxSPPTabID},
			"tabs": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":     "object",
					"required": []string{"id", "kind"},
					"properties": map[string]any{
						"id":             map[string]any{"type": "integer", "minimum": 1, "maximum": maxSPPTabID},
						"kind":           map[string]any{"type": "string", "enum": sppTabKinds},
						"filterStates":   map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "object"}},
						"searchValue":    map[string]any{"type": "string"},
						"filtersVisible": map[string]any{"type": "boolean"},
					},
				},
			},
		},
	},
	valid: validSPPTabsWorkspace,
}

func validSPPTabsWorkspace(raw json.RawMessage) bool {
	var value struct {
		Version  int
		ActiveID *int64
		Tabs     []struct {
			ID             int64
			Kind           string
			FilterStates   map[string]map[string]any
			SearchValue    string
			FiltersVisible bool
		}
	}
	if err := json.Unmarshal(raw, &value); err != nil || value.Version != 1 || value.Tabs == nil {
		return false
	}
	if value.ActiveID != nil && (*value.ActiveID < 1 || *value.ActiveID > maxSPPTabID) {
		return false
	}
	seen := make(map[int64]bool, len(value.Tabs))
	for _, tab := range value.Tabs {
		if tab.ID < 1 || tab.ID > maxSPPTabID || seen[tab.ID] || !slices.Contains(sppTabKinds, tab.Kind) {
			return false
		}
		seen[tab.ID] = true
	}
	return true
}
