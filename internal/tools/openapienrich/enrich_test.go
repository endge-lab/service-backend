package main

import (
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

const fixture = `openapi: 3.0.0
info:
  title: Test
  description: Из кода.
  version: 1.0.20
paths:
  /items:
    post:
      operationId: createItem
      requestBody:
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/item.Request"
      responses:
        "200":
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/item.Response"
components:
  securitySchemes:
    BearerAuth:
      type: apiKey
      name: Authorization
      in: header
  schemas:
    entities.Actor:
      type: object
      properties:
        id:
          type: string
          format: uuid
    item.Request:
      type: object
      required: [identity]
      properties:
        identity:
          type: string
        deletedAt:
          type: string
          format: date-time
    item.Response:
      type: object
      properties:
        identity:
          type: string
          description: Описание из кода.
          example: main
        createdBy:
          $ref: "#/components/schemas/entities.Actor"
`

// TestEnrichAddsDescriptionsWithoutOverwritingCode проверяет словарь, переопределения, версию и примеры.
func TestEnrichAddsDescriptionsWithoutOverwritingCode(t *testing.T) {
	var document yaml.Node
	if err := yaml.Unmarshal([]byte(fixture), &document); err != nil {
		t.Fatal(err)
	}
	words := glossary{
		Properties:      map[string]string{"identity": "Из словаря.", "deletedAt": "Время удаления.", "createdBy": "Автор."},
		Schemas:         map[string]map[string]string{"entities.*": {"id": "UUID пользователя."}},
		SecuritySchemes: map[string]string{"BearerAuth": "Bearer-токен."},
	}
	words.Info.Description = "Для агентов."
	result, err := enrich(&document, words, examples{}, "0.16.1")
	if err != nil {
		t.Fatal(err)
	}
	if result.DescribedProperties != result.Properties || result.Properties != 5 {
		t.Fatalf("described %d of %d, want 5 of 5; undescribed %v", result.DescribedProperties, result.Properties, result.Undescribed)
	}
	if result.ExampleOperations != 1 {
		t.Fatalf("example operations = %d, want 1", result.ExampleOperations)
	}
	raw, err := yaml.Marshal(&document)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, want := range []string{
		"version: 0.16.1",
		"Для агентов.",
		"description: Описание из кода.",
		"description: Bearer-токен.",
		"description: UUID пользователя.",
		"allOf:",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("enriched document misses %q:\n%s", want, text)
		}
	}
	if strings.Count(text, "description: Из словаря.") != 1 {
		t.Fatalf("glossary must describe only the request identity:\n%s", text)
	}
	if strings.Contains(text, "deletedAt: \"20") {
		t.Fatalf("synthesized example must skip optional deletedAt:\n%s", text)
	}
}
