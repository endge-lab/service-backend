package documents

import "testing"

// TestPatchInputJSONPreservesExplicitNullAndOmittedFields проверяет точную PATCH-семантику.
func TestPatchInputJSONPreservesExplicitNullAndOmittedFields(t *testing.T) {
	t.Parallel()
	input, err := NewPatchInputJSON([]byte(`{"description":null}`))
	if err != nil {
		t.Fatalf("NewPatchInputJSON: %v", err)
	}
	values, err := input.values()
	if err != nil {
		t.Fatalf("values: %v", err)
	}
	if value, exists := values["description"]; !exists || value != nil {
		t.Fatalf("description = %#v, exists = %v; explicit null must be preserved", value, exists)
	}
	if _, exists := values["displayName"]; exists {
		t.Fatal("omitted displayName unexpectedly appeared in patch")
	}
}
