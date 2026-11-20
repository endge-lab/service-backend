package user_state

import (
	"encoding/json"
	"math"
	"strings"
	"unicode"
)

type Definition struct {
	Key         string            `json:"key"`
	Description map[string]string `json:"description"`
	Schema      map[string]any    `json:"schema"`
	Default     any               `json:"default"`
	valid       func(json.RawMessage) bool
}

var Catalog = []Definition{
	{
		Key: "endge.ui.zoom",
		Description: map[string]string{
			"ru": "Масштаб интерфейса в процентах. Допустимы целые числа от 50 до 150.",
			"en": "Interface zoom as a percentage. Use an integer from 50 to 150.",
		},
		Schema:  map[string]any{"type": "integer", "minimum": 50, "maximum": 150},
		Default: 100,
		valid: func(raw json.RawMessage) bool {
			var value float64
			return json.Unmarshal(raw, &value) == nil && value >= 50 && value <= 150 && math.Trunc(value) == value
		},
	},
	{
		Key: "configurator.date-format",
		Description: map[string]string{
			"ru": "Формат времени в Configurator. human показывает относительное время, robot — дату и время.",
			"en": "Time display in Configurator. human shows relative times, robot shows the date and time.",
		},
		Schema:  map[string]any{"type": "string", "enum": []string{"human", "robot"}},
		Default: "human",
		valid: func(raw json.RawMessage) bool {
			var value string
			return json.Unmarshal(raw, &value) == nil && (value == "human" || value == "robot")
		},
	},
	sppTabsWorkspaceDefinition,
}

func validOperation(op Operation) bool {
	if len(op.Key) == 0 || len(op.Key) > 512 || strings.TrimSpace(op.Key) != op.Key ||
		strings.IndexFunc(op.Key, unicode.IsControl) >= 0 {
		return false
	}
	if op.Delete {
		return len(op.Value) == 0
	}
	if len(op.Value) > 60*1024 {
		return false
	}
	switch op.Key {
	case "appearance.theme", "appearance.locale", "appearance.timezone":
		var value *string
		return json.Unmarshal(op.Value, &value) == nil &&
			(value == nil && op.Key == "appearance.theme" || value != nil && len(*value) > 0 && len(*value) <= 256)
	}
	for _, definition := range Catalog {
		if definition.Key == op.Key {
			return definition.valid(op.Value)
		}
	}
	return json.Valid(op.Value)
}
