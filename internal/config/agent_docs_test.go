package config

import "testing"

func TestNormalizeAgentDocsURL(t *testing.T) {
	for input, want := range map[string]string{
		"":                               "",
		"  https://docs.endge.dev/ ":     "https://docs.endge.dev",
		"http://localhost:5174/endge":    "http://localhost:5174/endge",
		"https://docs.example.com/a/b//": "https://docs.example.com/a/b",
	} {
		got, err := normalizeAgentDocsURL(input)
		if err != nil || got != want {
			t.Fatalf("normalizeAgentDocsURL(%q) = %q, %v; want %q", input, got, err, want)
		}
	}
	for _, input := range []string{"docs.endge.dev", "ftp://docs", "https://docs?x=1", "/llms.txt"} {
		if _, err := normalizeAgentDocsURL(input); err == nil {
			t.Fatalf("normalizeAgentDocsURL(%q) must fail", input)
		}
	}
}
