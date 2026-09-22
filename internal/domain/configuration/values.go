package configuration

import (
	"fmt"
)

// ValidateValuesShape checks the storage-only configuration.values namespace.
// Semantic Type Registry validation remains owned by Endge Core.
func ValidateValuesShape(value any) error {
	configuration, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("configuration must be an object")
	}
	rawValues, exists := configuration["values"]
	if !exists || rawValues == nil {
		return nil
	}
	values, ok := rawValues.(map[string]any)
	if !ok {
		return fmt.Errorf("configuration.values must be an object")
	}
	for identity, rawCategory := range values {
		if !safeKey(identity) {
			return fmt.Errorf("configuration.values contains unsafe identity %q", identity)
		}
		category, ok := rawCategory.(map[string]any)
		if !ok {
			return fmt.Errorf("configuration.values.%s must be an object", identity)
		}
		for field := range category {
			if !safeKey(field) {
				return fmt.Errorf("configuration.values.%s contains unsafe field %q", identity, field)
			}
		}
	}
	return nil
}

func safeKey(value string) bool {
	return value != "" && value != "__proto__" && value != "prototype" && value != "constructor"
}
