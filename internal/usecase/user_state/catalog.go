package user_state

import (
	"encoding/json"
	"math"
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
}

func validOperation(op Operation) bool {
	for _, definition := range Catalog {
		if definition.Key == op.Key {
			if op.Delete {
				return len(op.Value) == 0
			}
			return definition.valid(op.Value)
		}
	}
	return false
}
