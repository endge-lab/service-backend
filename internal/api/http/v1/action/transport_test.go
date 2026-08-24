package action

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestActionResponseUsesSourceContract(t *testing.T) {
	payload, err := json.Marshal(Response{Source: "defineAction({})", SourceVersion: 1})
	if err != nil {
		t.Fatal(err)
	}
	text := string(payload)
	if !strings.Contains(text, `"source":"defineAction({})"`) || !strings.Contains(text, `"sourceVersion":1`) {
		t.Fatalf("source contract missing: %s", text)
	}
	for _, legacy := range []string{"definition", "input", "output"} {
		if strings.Contains(text, `"`+legacy+`"`) {
			t.Fatalf("legacy Action field %q is still serialized: %s", legacy, text)
		}
	}
}

