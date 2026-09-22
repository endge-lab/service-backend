package workspace_state

import "testing"

func TestActionSourceValidation(t *testing.T) {
	valid := map[string]any{
		"identity": "orders.save", "displayName": "Save", "source": "defineAction({})", "sourceVersion": float64(1),
	}
	if err := validateDocument("actions", valid); err != nil {
		t.Fatalf("valid Action Source rejected: %v", err)
	}
	for name, mutate := range map[string]func(map[string]any){
		"missing source":  func(value map[string]any) { delete(value, "source") },
		"empty source":    func(value map[string]any) { value["source"] = "  " },
		"missing version": func(value map[string]any) { delete(value, "sourceVersion") },
	} {
		t.Run(name, func(t *testing.T) {
			input := map[string]any{}
			for key, value := range valid {
				input[key] = value
			}
			mutate(input)
			if err := validateDocument("actions", input); err == nil {
				t.Fatal("invalid Action Source was accepted")
			}
		})
	}
}

func TestActionSourceSizeLimit(t *testing.T) {
	input := map[string]any{
		"identity": "orders.save", "displayName": "Save", "source": string(make([]byte, 8*1024*1024+1)), "sourceVersion": float64(1),
	}
	if err := validateDocument("actions", input); err == nil {
		t.Fatal("oversized Action Source was accepted")
	}
}
