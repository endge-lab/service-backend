package workspace_state

import "testing"

func TestVocabRejectsLegacyManualAuthMode(t *testing.T) {
	input := map[string]any{
		"identity":    "airlines",
		"displayName": "Airlines",
		"authMode":    "manual",
	}
	if err := validateDocument("vocabs", input); err == nil {
		t.Fatal("legacy manual vocab auth mode was accepted")
	}
}

func TestAuthProfileAllowsPluginAdapterAndRejectsSecretConfig(t *testing.T) {
	input := map[string]any{
		"identity":    "plugin-auth",
		"displayName": "Plugin auth",
		"adapterId":   "company-plugin",
		"credentials": map[string]any{"secret": "{COMPANY_SECRET}"},
	}
	if err := validateDocument("auth-profiles", input); err != nil {
		t.Fatalf("plugin adapter rejected: %v", err)
	}
	if err := validateSecrets(map[string]any{"config": map[string]any{"password": "raw"}}); err == nil {
		t.Fatal("secret AuthProfile config was accepted")
	}
}
