package mcp

import (
	"reflect"
	"slices"
	"strings"

	"github.com/endge-lab/service-backend/internal/usecase/documents"
)

// formatVersionFields — имена полей версии формата документа в REST DTO.
var formatVersionFields = []string{"sourceVersion", "modelVersion"}

// commonCreateFields — поля тела создания, которые описаны в инструменте отдельно
// и не повторяются в списке дополнительных обязательных полей коллекции.
var commonCreateFields = []string{"identity", "displayName", "source", "sourceVersion", "modelVersion"}

// FormatVersion — поле версии формата документа коллекции, выведенное из её CreateRequest.
// Описание инструментов строится из него, поэтому не расходится с проверкой DTO.
type FormatVersion struct {
	// Field — sourceVersion или modelVersion; пусто, если у коллекции нет версии формата.
	Field string
	// Required — поле обязательно при создании (validate:"required").
	Required bool
	// WithSource — DTO допускает пропуск, но use case требует поле вместе с source (source_version_invalid).
	WithSource bool
	// Exact — сервер принимает только это значение (validate:"eq=…").
	Exact bool
	// Current — текущая версия формата: значение example-тега DTO. Его пишет Configurator
	// при создании документа и его же принимает компилятор Core.
	Current string
	// Also — другие обязательные поля тела создания коллекции, например contractVersion=1
	// у computations или storeIdentity у updates.
	Also []string
}

// formatVersionOf ищет поле версии и дополнительные обязательные поля в DTO создания
// (включая встроенные структуры).
func formatVersionOf(collection string, request reflect.Type) FormatVersion {
	version := FormatVersion{Also: extraRequiredFields(request)}
	field, ok := findVersionField(request)
	if !ok {
		return version
	}
	version.Field = strings.Split(field.Tag.Get("json"), ",")[0]
	version.Current = field.Tag.Get("example")
	for _, rule := range strings.Split(field.Tag.Get("validate"), ",") {
		switch {
		case rule == "required":
			version.Required = true
		case strings.HasPrefix(rule, "eq="):
			version.Exact = true
			version.Current = strings.TrimPrefix(rule, "eq=")
		}
	}
	if version.Field == "sourceVersion" && !version.Required && documents.RequiresSourceVersion(collection) {
		version.WithSource = true
	}
	return version
}

func findVersionField(request reflect.Type) (reflect.StructField, bool) {
	var found reflect.StructField
	ok := false
	walkFields(request, func(field reflect.StructField, name string) bool {
		if ok || !slices.Contains(formatVersionFields, name) {
			return true
		}
		found, ok = field, true
		return false
	})
	return found, ok
}

// extraRequiredFields — обязательные поля DTO, кроме общих: «storeIdentity», «contractVersion=1».
func extraRequiredFields(request reflect.Type) []string {
	var fields []string
	walkFields(request, func(field reflect.StructField, name string) bool {
		if slices.Contains(commonCreateFields, name) || !slices.Contains(strings.Split(field.Tag.Get("validate"), ","), "required") {
			return true
		}
		if strings.HasSuffix(name, "Version") && field.Tag.Get("example") != "" {
			name += "=" + field.Tag.Get("example")
		}
		fields = append(fields, name)
		return true
	})
	return fields
}

// walkFields обходит поля структуры с JSON-именем, включая встроенные; visit возвращает false, чтобы остановиться.
func walkFields(request reflect.Type, visit func(field reflect.StructField, name string) bool) bool {
	for request.Kind() == reflect.Pointer {
		request = request.Elem()
	}
	if request.Kind() != reflect.Struct {
		return true
	}
	for i := range request.NumField() {
		field := request.Field(i)
		if field.Anonymous {
			if !walkFields(field.Type, visit) {
				return false
			}
			continue
		}
		name := strings.Split(field.Tag.Get("json"), ",")[0]
		if name == "" || name == "-" {
			continue
		}
		if !visit(field, name) {
			return false
		}
	}
	return true
}

// describe — строка для описания инструмента, например «sourceVersion=1 (обязательно); ещё обязательно: contractVersion=1».
func (v FormatVersion) describe() string {
	text := "нет поля версии"
	if v.Field != "" {
		text = v.Field
		if v.Current != "" {
			text += "=" + v.Current
		}
		switch {
		case v.Required && v.Exact:
			text += " (обязательно, другое значение — ошибка)"
		case v.Required:
			text += " (обязательно)"
		case v.WithSource:
			text += " (обязательно вместе с source)"
		default:
			text += " (необязательно)"
		}
	}
	if len(v.Also) > 0 {
		text += "; ещё обязательно: " + strings.Join(v.Also, ", ")
	}
	return text
}

// formatVersionSummary перечисляет поле версии формата по коллекциям в порядке collections,
// пропуская коллекции из skip.
func formatVersionSummary(versions map[string]FormatVersion, collections []string, skip ...string) string {
	lines := make([]string, 0, len(collections))
	for _, collection := range collections {
		version, ok := versions[collection]
		if !ok || slices.Contains(skip, collection) {
			continue
		}
		lines = append(lines, "- "+collection+": "+version.describe())
	}
	return strings.Join(lines, "\n")
}
