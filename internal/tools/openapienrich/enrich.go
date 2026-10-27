package main

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

// glossary — вручную поддерживаемый словарь docs/openapi-glossary.yaml.
type glossary struct {
	Info struct {
		// Description дописывается к info.description из аннотаций cmd/main.go.
		Description string `yaml:"description"`
	} `yaml:"info"`
	SecuritySchemes map[string]string `yaml:"securitySchemes"`
	// Properties — описание свойства по имени для всех схем.
	Properties map[string]string `yaml:"properties"`
	// Schemas — переопределения для схемы (точное имя или шаблон path.Match, например composition.*):
	// ключ "" — описание самой схемы, остальные — описания свойств.
	Schemas map[string]map[string]string `yaml:"schemas"`
}

// examples — примеры тел запросов и ответов docs/openapi-examples.yaml по operationId.
type examples struct {
	Operations map[string]struct {
		Request  *yaml.Node `yaml:"request"`
		Response *yaml.Node `yaml:"response"`
	} `yaml:"operations"`
}

// report — итог обогащения для журнала make docs и проверки покрытия.
type report struct {
	Properties          int
	DescribedProperties int
	Undescribed         []string
	UnusedGlossary      []string
	UnusedOverrides     []string
	UnusedExamples      []string
	BodyOperations      int
	ExampleOperations   int
}

type enricher struct {
	glossary      glossary
	examples      examples
	version       string
	schemas       *yaml.Node
	usedNames     map[string]bool // имена свойств, которые встречаются в схемах
	usedOverrides map[string]bool
	usedExamples  map[string]bool
	report        report
}

func enrich(document *yaml.Node, words glossary, samples examples, version string) (report, error) {
	if document.Kind != yaml.DocumentNode || len(document.Content) != 1 {
		return report{}, fmt.Errorf("unexpected YAML document")
	}
	root := document.Content[0]
	components := mapGet(root, "components")
	e := &enricher{
		glossary: words, examples: samples, version: version,
		schemas:       mapGet(components, "schemas"),
		usedNames:     map[string]bool{},
		usedOverrides: map[string]bool{},
		usedExamples:  map[string]bool{},
	}
	if e.schemas == nil {
		return report{}, fmt.Errorf("components.schemas is missing")
	}
	if err := e.enrichInfo(root); err != nil {
		return report{}, err
	}
	if err := e.enrichSecurity(components); err != nil {
		return report{}, err
	}
	e.enrichSchemas()
	if err := e.enrichOperations(mapGet(root, "paths")); err != nil {
		return report{}, err
	}
	e.collectUnused()
	return e.report, nil
}

func (e *enricher) enrichInfo(root *yaml.Node) error {
	info := mapGet(root, "info")
	if info == nil {
		return fmt.Errorf("info is missing")
	}
	if e.version == "" {
		return fmt.Errorf("APP_VERSION is empty")
	}
	mapSet(info, "version", stringNode(e.version))
	if appendix := strings.TrimSpace(e.glossary.Info.Description); appendix != "" {
		description := ""
		if current := mapGet(info, "description"); current != nil {
			description = strings.TrimSpace(current.Value)
		}
		if !strings.Contains(description, appendix) {
			description = strings.TrimSpace(description + "\n\n" + appendix)
		}
		mapSet(info, "description", textNode(description))
	}
	return nil
}

func (e *enricher) enrichSecurity(components *yaml.Node) error {
	schemes := mapGet(components, "securitySchemes")
	for name, text := range e.glossary.SecuritySchemes {
		scheme := mapGet(schemes, name)
		if scheme == nil {
			return fmt.Errorf("security scheme %q from glossary is missing in OpenAPI", name)
		}
		if mapGet(scheme, "description") == nil {
			mapSet(scheme, "description", textNode(strings.TrimSpace(text)))
		}
	}
	return nil
}

func (e *enricher) enrichSchemas() {
	for index := 0; index+1 < len(e.schemas.Content); index += 2 {
		name := e.schemas.Content[index].Value
		schema := e.schemas.Content[index+1]
		if text, ok := e.override(name, ""); ok && mapGet(schema, "description") == nil {
			mapSet(schema, "description", textNode(text))
		}
		e.enrichProperties(name, schema)
	}
}

// enrichProperties описывает свойства схемы, включая вложенные inline-объекты и элементы массивов.
func (e *enricher) enrichProperties(schemaName string, schema *yaml.Node) {
	if schema == nil || schema.Kind != yaml.MappingNode {
		return
	}
	if properties := mapGet(schema, "properties"); properties != nil {
		for index := 0; index+1 < len(properties.Content); index += 2 {
			propertyName := properties.Content[index].Value
			property := properties.Content[index+1]
			e.report.Properties++
			e.usedNames[propertyName] = true
			if mapGet(property, "description") == nil {
				text, ok := e.override(schemaName, propertyName)
				if !ok {
					text, ok = e.glossary.Properties[propertyName]
				}
				if ok && strings.TrimSpace(text) != "" {
					properties.Content[index+1] = describeProperty(property, strings.TrimSpace(text))
					property = properties.Content[index+1]
				}
			}
			if mapGet(property, "description") != nil {
				e.report.DescribedProperties++
			} else {
				e.report.Undescribed = append(e.report.Undescribed, schemaName+"."+propertyName)
			}
			e.enrichProperties(schemaName, property)
		}
	}
	e.enrichProperties(schemaName, mapGet(schema, "items"))
	if additional := mapGet(schema, "additionalProperties"); additional != nil && additional.Kind == yaml.MappingNode {
		e.enrichProperties(schemaName, additional)
	}
}

// override ищет описание в schemas: точное имя схемы важнее шаблона.
func (e *enricher) override(schemaName, propertyName string) (string, bool) {
	if values, ok := e.glossary.Schemas[schemaName]; ok {
		if text, ok := values[propertyName]; ok {
			e.usedOverrides[schemaName+"#"+propertyName] = true
			return strings.TrimSpace(text), true
		}
	}
	patterns := make([]string, 0, len(e.glossary.Schemas))
	for pattern := range e.glossary.Schemas {
		if strings.ContainsAny(pattern, "*?[") {
			patterns = append(patterns, pattern)
		}
	}
	sort.Strings(patterns)
	for _, pattern := range patterns {
		if matched, _ := path.Match(pattern, schemaName); !matched {
			continue
		}
		if text, ok := e.glossary.Schemas[pattern][propertyName]; ok {
			e.usedOverrides[pattern+"#"+propertyName] = true
			return strings.TrimSpace(text), true
		}
	}
	return "", false
}

// describeProperty добавляет description; ссылку $ref оборачивает в allOf, так как в OpenAPI 3.0
// соседние с $ref ключи игнорируются.
func describeProperty(property *yaml.Node, text string) *yaml.Node {
	if property.Kind != yaml.MappingNode {
		return property
	}
	if ref := mapGet(property, "$ref"); ref != nil && len(property.Content) == 2 {
		wrapped := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		mapSet(wrapped, "description", textNode(text))
		mapSet(wrapped, "allOf", &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Content: []*yaml.Node{property}})
		return wrapped
	}
	description := textNode(text)
	property.Content = append([]*yaml.Node{stringNode("description"), description}, property.Content...)
	return property
}

var operationMethods = []string{"get", "put", "post", "delete", "patch"}

func (e *enricher) enrichOperations(paths *yaml.Node) error {
	if paths == nil {
		return fmt.Errorf("paths are missing")
	}
	for index := 0; index+1 < len(paths.Content); index += 2 {
		item := paths.Content[index+1]
		for _, method := range operationMethods {
			operation := mapGet(item, method)
			if operation == nil {
				continue
			}
			id := ""
			if node := mapGet(operation, "operationId"); node != nil {
				id = node.Value
			}
			explicit := e.examples.Operations[id]
			if _, ok := e.examples.Operations[id]; ok {
				e.usedExamples[id] = true
			}
			var bodies []*yaml.Node
			var samples []*yaml.Node
			hasRequest := false
			if media := jsonMedia(mapGet(operation, "requestBody")); media != nil {
				bodies, samples = append(bodies, media), append(samples, explicit.Request)
				hasRequest = true
			}
			if media := successMedia(mapGet(operation, "responses")); media != nil {
				bodies, samples = append(bodies, media), append(samples, explicit.Response)
			}
			if len(bodies) == 0 {
				continue
			}
			e.report.BodyOperations++
			described := false
			for bodyIndex, media := range bodies {
				if mapGet(media, "example") != nil || mapGet(media, "examples") != nil {
					described = true
					continue
				}
				sample := samples[bodyIndex]
				if sample == nil {
					sample = e.synthesize(mapGet(media, "schema"), 0, map[string]bool{})
					if bodyIndex == 0 && hasRequest && method == "patch" {
						sample = partialExample(sample)
					}
				}
				if sample != nil {
					mapSet(media, "example", sample)
					described = true
				}
			}
			if described {
				e.report.ExampleOperations++
			}
		}
	}
	return nil
}

func jsonMedia(container *yaml.Node) *yaml.Node {
	content := mapGet(container, "content")
	if content == nil {
		return nil
	}
	return mapGet(content, "application/json")
}

func successMedia(responses *yaml.Node) *yaml.Node {
	if responses == nil {
		return nil
	}
	for index := 0; index+1 < len(responses.Content); index += 2 {
		if strings.HasPrefix(responses.Content[index].Value, "2") {
			if media := jsonMedia(responses.Content[index+1]); media != nil {
				return media
			}
		}
	}
	return nil
}

const maxExampleDepth = 6

// exampleSkipped — свойства, которые не попадают в собранные примеры: их значения зависят от состояния
// (удалён, отозван) или от коллекции (папки), и общий example ввёл бы агента в заблуждение.
var exampleSkipped = map[string]bool{
	"deletedAt": true, "revokedAt": true, "revokedBy": true,
	"folderIdentity": true, "workspaceFolderIdentity": true, "managedById": true,
}

// patchExampleFields — поля, которые показывает пример частичного изменения (PATCH).
var patchExampleFields = map[string]bool{
	"displayName": true, "description": true, "source": true, "sourceVersion": true,
	"name": true, "enabled": true, "role": true, "settings": true,
}

// synthesize собирает пример тела из example/enum схем. Свойства без примера включаются,
// только если они обязательны; формат date-time и uuid получает типовое значение.
func (e *enricher) synthesize(schema *yaml.Node, depth int, visiting map[string]bool) *yaml.Node {
	if schema == nil || depth > maxExampleDepth {
		return nil
	}
	if example := mapGet(schema, "example"); example != nil {
		return example
	}
	if ref := mapGet(schema, "$ref"); ref != nil {
		name := strings.TrimPrefix(ref.Value, "#/components/schemas/")
		if visiting[name] {
			return nil
		}
		visiting[name] = true
		defer delete(visiting, name)
		return e.synthesize(mapGet(e.schemas, name), depth+1, visiting)
	}
	if allOf := mapGet(schema, "allOf"); allOf != nil && len(allOf.Content) == 1 {
		return e.synthesize(allOf.Content[0], depth, visiting)
	}
	if enum := mapGet(schema, "enum"); enum != nil && len(enum.Content) > 0 {
		return enum.Content[0]
	}
	kind := ""
	if node := mapGet(schema, "type"); node != nil {
		kind = node.Value
	}
	if kind == "" && mapGet(schema, "properties") != nil {
		kind = "object"
	}
	switch kind {
	case "object":
		properties := mapGet(schema, "properties")
		if properties == nil {
			return nil
		}
		required := map[string]bool{}
		if list := mapGet(schema, "required"); list != nil {
			for _, item := range list.Content {
				required[item.Value] = true
			}
		}
		result := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		for index := 0; index+1 < len(properties.Content); index += 2 {
			name := properties.Content[index].Value
			if exampleSkipped[name] && !required[name] {
				continue
			}
			value := e.synthesize(properties.Content[index+1], depth+1, visiting)
			if value == nil && required[name] {
				value = placeholder(name, properties.Content[index+1])
			}
			if value != nil {
				mapSet(result, name, value)
			}
		}
		if len(result.Content) == 0 {
			return nil
		}
		return result
	case "array":
		item := e.synthesize(mapGet(schema, "items"), depth+1, visiting)
		if item == nil {
			return nil
		}
		return &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Content: []*yaml.Node{item}}
	case "string":
		switch format := mapGet(schema, "format"); {
		case format != nil && format.Value == "date-time":
			return stringNode("2026-08-04T10:00:00Z")
		case format != nil && format.Value == "uuid":
			return stringNode("550e8400-e29b-41d4-a716-446655440000")
		}
	}
	return nil
}

// placeholder подставляет типовое значение обязательного свойства без example.
func placeholder(name string, schema *yaml.Node) *yaml.Node {
	kind := ""
	if node := mapGet(schema, "type"); node != nil {
		kind = node.Value
	}
	switch kind {
	case "integer", "number":
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: "1"}
	case "boolean":
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "false"}
	case "array":
		return &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Style: yaml.FlowStyle}
	case "object":
		return &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Style: yaml.FlowStyle}
	default:
		if name == "id" || strings.HasSuffix(name, "Id") {
			return stringNode("550e8400-e29b-41d4-a716-446655440000")
		}
		return stringNode("string")
	}
}

func (e *enricher) collectUnused() {
	for name := range e.glossary.Properties {
		if !e.usedNames[name] {
			e.report.UnusedGlossary = append(e.report.UnusedGlossary, name)
		}
	}
	for schema, values := range e.glossary.Schemas {
		for property := range values {
			if !e.usedOverrides[schema+"#"+property] {
				e.report.UnusedOverrides = append(e.report.UnusedOverrides, schema+"#"+property)
			}
		}
	}
	for id := range e.examples.Operations {
		if !e.usedExamples[id] {
			e.report.UnusedExamples = append(e.report.UnusedExamples, id)
		}
	}
	sort.Strings(e.report.Undescribed)
	sort.Strings(e.report.UnusedGlossary)
	sort.Strings(e.report.UnusedOverrides)
	sort.Strings(e.report.UnusedExamples)
}

func mapGet(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for index := 0; index+1 < len(node.Content); index += 2 {
		if node.Content[index].Value == key {
			return node.Content[index+1]
		}
	}
	return nil
}

func mapSet(node *yaml.Node, key string, value *yaml.Node) {
	for index := 0; index+1 < len(node.Content); index += 2 {
		if node.Content[index].Value == key {
			node.Content[index+1] = value
			return
		}
	}
	node.Content = append(node.Content, stringNode(key), value)
}

func stringNode(value string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
}

// textNode хранит многострочный текст литеральным блоком, чтобы абзацы и ссылки читались в YAML.
func textNode(value string) *yaml.Node {
	node := stringNode(value)
	if strings.Contains(value, "\n") {
		node.Style = yaml.LiteralStyle
	}
	return node
}

// partialExample оставляет в примере PATCH только типично изменяемые поля; если их нет — пример целиком.
func partialExample(sample *yaml.Node) *yaml.Node {
	if sample == nil || sample.Kind != yaml.MappingNode {
		return sample
	}
	partial := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	for index := 0; index+1 < len(sample.Content); index += 2 {
		if patchExampleFields[sample.Content[index].Value] {
			partial.Content = append(partial.Content, sample.Content[index], sample.Content[index+1])
		}
	}
	if len(partial.Content) == 0 {
		return sample
	}
	return partial
}
