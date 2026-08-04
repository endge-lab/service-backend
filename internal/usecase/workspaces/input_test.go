package workspaces

import "testing"

// TestPatchInputJSONPreservesNullableFields проверяет null и пустой список integrations в PATCH.
func TestPatchInputJSONPreservesNullableFields(t *testing.T) {
	t.Parallel()
	input, err := NewPatchInputJSON([]byte(`{"description":null,"installedIntegrations":[]}`))
	if err != nil {
		t.Fatalf("NewPatchInputJSON: %v", err)
	}
	values, err := input.values()
	if err != nil {
		t.Fatalf("values: %v", err)
	}
	if value, exists := values["description"]; !exists || value != nil {
		t.Fatalf("description = %#v, exists = %v", value, exists)
	}
	if integrations, exists := values["installedIntegrations"].([]any); !exists || len(integrations) != 0 {
		t.Fatalf("installedIntegrations = %#v", values["installedIntegrations"])
	}
}
