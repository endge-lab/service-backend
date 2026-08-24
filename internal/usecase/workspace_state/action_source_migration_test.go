package workspace_state

import (
	"os"
	"strings"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

func TestPortableActionSourceMigration(t *testing.T) {
	bundle := portableBundleWithActions(t, []map[string]any{{
		"identity": "orders.save", "displayName": "Save", "definition": map[string]any{"nodes": []any{}, "edges": []any{}},
	}})
	result := normalizePortableBundle(&bundle)
	action := bundle.Documents["actions"][0]
	if result.MigratedLegacyActions != 1 || !strings.Contains(stringField(action, "source"), "defineAction") {
		t.Fatalf("empty legacy action was not migrated: result=%+v action=%+v", result, action)
	}
	if _, exists := action["definition"]; exists {
		t.Fatal("legacy definition was retained")
	}
}

func TestPortableActionSourceMigrationRejectsNonEmptyFlow(t *testing.T) {
	bundle := portableBundleWithActions(t, []map[string]any{{
		"identity": "orders.save", "displayName": "Save", "definition": map[string]any{"nodes": []any{map[string]any{"id": "one"}}},
	}})
	result := normalizePortableBundle(&bundle)
	if len(result.NonEmptyLegacyActions) != 1 || result.NonEmptyLegacyActions[0] != "orders.save" {
		t.Fatalf("non-empty legacy action was not rejected: %+v", result)
	}
}

func portableBundleWithActions(t *testing.T, actions []map[string]any) entities.PortableBundle {
	t.Helper()
	return entities.PortableBundle{
		Workspace: map[string]any{"identity": "workspace", "displayName": "Workspace"},
		Documents: map[string][]map[string]any{"actions": actions},
	}
}

func TestActionSourceMigrationGuardsNonEmptyFlow(t *testing.T) {
	data, err := os.ReadFile("../../../migrations/000053_action_source.sql")
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	for _, contract := range []string{"non-empty legacy Flow", "jsonb_array_length", "defineAction", "sourceVersion"} {
		if !strings.Contains(source, contract) {
			t.Fatalf("migration does not contain %q guard/contract", contract)
		}
	}
}
