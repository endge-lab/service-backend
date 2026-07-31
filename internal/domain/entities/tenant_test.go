package entities

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestDefaultEndgeConfigurationContributionJSON(t *testing.T) {
	t.Parallel()

	contribution := DefaultEndgeConfigurationContribution()

	if contribution.Patch == nil {
		t.Fatal("default contribution patch must be an empty object, not nil")
	}

	encoded, err := json.Marshal(contribution)
	if err != nil {
		t.Fatalf("marshal default contribution: %v", err)
	}

	const want = `{"mode":"inherit","patch":{}}`
	if got := string(encoded); got != want {
		t.Fatalf("default contribution JSON = %s, want %s", got, want)
	}
}

func TestEndgeConfigurationContributionRoundTripsPatch(t *testing.T) {
	t.Parallel()

	original := EndgeConfigurationContribution{
		Mode: EndgeConfigurationContributionModeInherit,
		Patch: map[string]json.RawMessage{
			EndgeConfigurationPatchKeyDefaultTheme: json.RawMessage(`{"op":"set","value":"tenant-brand"}`),
		},
	}

	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal contribution: %v", err)
	}

	var decoded EndgeConfigurationContribution
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal contribution: %v", err)
	}

	if decoded.Mode != EndgeConfigurationContributionModeInherit {
		t.Fatalf("mode = %q, want %q", decoded.Mode, EndgeConfigurationContributionModeInherit)
	}
	if got := string(decoded.Patch[EndgeConfigurationPatchKeyDefaultTheme]); got != `{"op":"set","value":"tenant-brand"}` {
		t.Fatalf("default theme patch = %s", got)
	}
	if decoded.Value != nil {
		t.Fatal("inherit contribution must not gain a replace value")
	}
}

func TestEndgeConfigurationContributionRoundTripsReplaceValue(t *testing.T) {
	t.Parallel()

	replaceConfiguration := DefaultEndgeConfiguration()
	original := EndgeConfigurationContribution{
		Mode:  EndgeConfigurationContributionModeReplace,
		Patch: map[string]json.RawMessage{},
		Value: &replaceConfiguration,
	}

	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal contribution: %v", err)
	}

	var decoded EndgeConfigurationContribution
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal contribution: %v", err)
	}

	if decoded.Mode != EndgeConfigurationContributionModeReplace {
		t.Fatalf("mode = %q, want %q", decoded.Mode, EndgeConfigurationContributionModeReplace)
	}
	if decoded.Value == nil {
		t.Fatal("replace contribution value must round-trip")
	}
	if decoded.Value.DefaultTheme != replaceConfiguration.DefaultTheme {
		t.Fatalf("replace default theme = %q, want %q", decoded.Value.DefaultTheme, replaceConfiguration.DefaultTheme)
	}
}

func TestTenantEntityScopeAndFolderType(t *testing.T) {
	t.Parallel()

	if FolderEntityTypeTenants != "tenants" {
		t.Fatalf("tenant folder entity type = %q", FolderEntityTypeTenants)
	}

	tenantType := reflect.TypeOf(RTenant{})
	for _, forbiddenField := range []string{"ProjectID", "EnvironmentID", "EffectiveConfiguration"} {
		if _, exists := tenantType.FieldByName(forbiddenField); exists {
			t.Errorf("RTenant must not contain %s", forbiddenField)
		}
	}
}
