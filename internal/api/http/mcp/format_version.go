package mcp

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/endge-lab/service-backend/internal/usecase/documents"
)

// formatVersionFields — имена полей версии формата документа в REST DTO.
var formatVersionFields = []string{"sourceVersion", "modelVersion"}

// FormatVersion — поле версии формата документа коллекции, выведенное из её CreateRequest.
// Описание инструментов строится из него, поэтому не расходится с проверкой DTO.
type FormatVersion struct {
	// Field — sourceVersion или modelVersion; пусто, если у коллекции нет версии формата.
	Field string
	// Required — поле обязательно при создании (validate:"required").
	Required bool
	// WithSource — DTO допускает пропуск, но use case требует поле вместе с source (source_version_invalid).
	WithSource bool
	// Rule — допустимые значения: «ровно 2», «целое > 0», «целое ≥ 1».
	Rule string
	// Example — значение из example-тега DTO.
	Example string
}

// formatVersionOf ищет поле версии в DTO создания (включая встроенные структуры).
func formatVersionOf(collection string, request reflect.Type) FormatVersion {
	field, ok := findVersionField(request)
	if !ok {
		return FormatVersion{}
	}
	name := strings.Split(field.Tag.Get("json"), ",")[0]
	rules := strings.Split(field.Tag.Get("validate"), ",")
	version := FormatVersion{Field: name, Example: field.Tag.Get("example"), Rule: "целое > 0"}
	for _, rule := range rules {
		switch {
		case rule == "required":
			version.Required = true
		case strings.HasPrefix(rule, "eq="):
			version.Rule = "ровно " + strings.TrimPrefix(rule, "eq=")
		case strings.HasPrefix(rule, "gt="):
			version.Rule = "целое > " + strings.TrimPrefix(rule, "gt=")
		case strings.HasPrefix(rule, "min="):
			version.Rule = "целое ≥ " + strings.TrimPrefix(rule, "min=")
		}
	}
	if name == "sourceVersion" && !version.Required && documents.RequiresSourceVersion(collection) {
		version.WithSource = true
	}
	return version
}

func findVersionField(request reflect.Type) (reflect.StructField, bool) {
	for request.Kind() == reflect.Pointer {
		request = request.Elem()
	}
	if request.Kind() != reflect.Struct {
		return reflect.StructField{}, false
	}
	for i := range request.NumField() {
		field := request.Field(i)
		if field.Anonymous {
			if nested, ok := findVersionField(field.Type); ok {
				return nested, true
			}
			continue
		}
		name := strings.Split(field.Tag.Get("json"), ",")[0]
		for _, candidate := range formatVersionFields {
			if name == candidate {
				return field, true
			}
		}
	}
	return reflect.StructField{}, false
}

// describe — строка для описания инструмента: «modelVersion (обязательно, целое > 0, например 1)».
func (v FormatVersion) describe() string {
	if v.Field == "" {
		return "нет поля версии"
	}
	parts := []string{"необязательно"}
	switch {
	case v.Required:
		parts[0] = "обязательно"
	case v.WithSource:
		parts[0] = "обязательно вместе с source"
	}
	parts = append(parts, v.Rule)
	if v.Example != "" && !strings.HasPrefix(v.Rule, "ровно") {
		parts = append(parts, "обычно "+v.Example)
	}
	return fmt.Sprintf("%s (%s)", v.Field, strings.Join(parts, ", "))
}

// formatVersionSummary перечисляет поле версии формата по коллекциям в порядке collections.
func formatVersionSummary(versions map[string]FormatVersion, collections []string) string {
	lines := make([]string, 0, len(collections))
	for _, collection := range collections {
		version, ok := versions[collection]
		if !ok {
			continue
		}
		lines = append(lines, "- "+collection+": "+version.describe())
	}
	return strings.Join(lines, "\n")
}
