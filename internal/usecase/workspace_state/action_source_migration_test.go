package workspace_state

import (
	"os"
	"strings"
	"testing"
)

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
