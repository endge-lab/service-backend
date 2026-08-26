package domainversion

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

func TestComputeIsStableForPortableOrderingAndTargetLocalFields(t *testing.T) {
	left := entities.PortableBundle{
		Kind: "workspace-snapshot", SchemaVersion: 1,
		Workspace: map[string]any{"identity": "source", "displayName": "Domain", "configuration": map[string]any{"b": 2, "a": 1}, "state": map[string]any{"id": "source-id"}},
		Documents: map[string][]map[string]any{
			"queries": {
				{"identity": "b", "source": "return 2", "state": map[string]any{"id": "b-id"}},
				{"identity": "a", "source": "return 1"},
			},
		},
		InstalledIntegrations: []map[string]any{{"identity": "source-only", "version": "1"}},
	}
	right := entities.PortableBundle{
		Kind: "workspace-snapshot", SchemaVersion: 1,
		Workspace: map[string]any{"configuration": map[string]any{"a": 1, "b": 2}, "displayName": "Domain", "identity": "target"},
		Documents: map[string][]map[string]any{
			"queries": {
				{"source": "return 1", "identity": "a"},
				{"source": "return 2", "identity": "b"},
			},
		},
		InstalledIntegrations: []map[string]any{{"identity": "target-only", "version": "9"}},
	}

	leftVersion, err := Compute(left)
	if err != nil {
		t.Fatal(err)
	}
	rightVersion, err := Compute(right)
	if err != nil {
		t.Fatal(err)
	}
	if leftVersion != rightVersion {
		t.Fatalf("portable equivalent domains must match: %s != %s", leftVersion, rightVersion)
	}
	if !strings.HasPrefix(leftVersion, "dv2:sha256:") {
		t.Fatalf("current domain version must use dv2: %s", leftVersion)
	}
}

func TestComputeCanonicalizesLegacyActionAndVocabBeforeHashing(t *testing.T) {
	legacy := entities.PortableBundle{
		Kind:          "workspace-snapshot",
		SchemaVersion: 1,
		Workspace: map[string]any{
			"displayName": "Domain", "configuration": map[string]any{},
		},
		Documents: map[string][]map[string]any{
			"actions": {{
				"identity": "orders.open", "definition": map[string]any{"nodes": []any{}, "edges": []any{}},
			}},
			"vocabs": {{
				"identity": "airlines", "mode": "external_payload", "baseApiUrl": "{ENDPOINT_VOCABS_SERVICE}", "authMode": "inherit",
			}},
		},
	}
	canonical, report, err := Canonicalize(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if report.MigratedLegacyActions != 1 || report.MigratedLegacyVocabs != 1 || !report.SFCEditingDefaultsAdded {
		t.Fatalf("legacy representations were not fully canonicalized: %+v", report)
	}

	legacyVersion, err := Compute(legacy)
	if err != nil {
		t.Fatal(err)
	}
	canonicalVersion, err := Compute(canonical)
	if err != nil {
		t.Fatal(err)
	}
	if legacyVersion != canonicalVersion {
		t.Fatalf("legacy and canonical representations must have one dv2 identity: %s != %s", legacyVersion, canonicalVersion)
	}

	second, secondReport, err := Canonicalize(canonical)
	if err != nil {
		t.Fatal(err)
	}
	firstRaw, _ := json.Marshal(canonical)
	secondRaw, _ := json.Marshal(second)
	if string(firstRaw) != string(secondRaw) {
		t.Fatalf("canonicalization is not idempotent:\n%s\n%s", firstRaw, secondRaw)
	}
	if secondReport.MigratedLegacyActions != 0 || secondReport.MigratedLegacyVocabs != 0 || secondReport.SFCEditingDefaultsAdded {
		t.Fatalf("second canonicalization still changed the bundle: %+v", secondReport)
	}
}

func TestAttachProducesRoundTripStableCanonicalSnapshot(t *testing.T) {
	exported := entities.PortableBundle{
		Kind:          "workspace-snapshot",
		SchemaVersion: 1,
		Workspace:     map[string]any{"identity": "source", "displayName": "Domain"},
		Documents: map[string][]map[string]any{
			"actions": {{"identity": "orders.open", "definition": map[string]any{"nodes": []any{}}}},
		},
	}
	if err := Attach(&exported); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(exported.DomainVersion, "dv2:sha256:") {
		t.Fatalf("exported snapshot does not use dv2: %q", exported.DomainVersion)
	}
	if _, exists := exported.Documents["actions"][0]["definition"]; exists {
		t.Fatal("export retained legacy Action definition")
	}
	if strings.TrimSpace(text(exported.Documents["actions"][0]["source"])) == "" {
		t.Fatal("export did not emit canonical Action source")
	}

	raw, err := json.Marshal(exported)
	if err != nil {
		t.Fatal(err)
	}
	var imported entities.PortableBundle
	if err = json.Unmarshal(raw, &imported); err != nil {
		t.Fatal(err)
	}
	verified, err := ComputeForDeclaredVersion(imported, imported.DomainVersion)
	if err != nil {
		t.Fatal(err)
	}
	if verified != exported.DomainVersion {
		t.Fatalf("same exported file failed import verification: %s != %s", verified, exported.DomainVersion)
	}
	if err = Attach(&imported); err != nil {
		t.Fatal(err)
	}
	if imported.DomainVersion != exported.DomainVersion {
		t.Fatalf("domain version changed after export/import round trip: %s != %s", imported.DomainVersion, exported.DomainVersion)
	}
}

func TestComputeForDeclaredVersionKeepsDV1ValidationCompatibility(t *testing.T) {
	bundle := entities.PortableBundle{
		Kind:          "workspace-snapshot",
		SchemaVersion: 1,
		Workspace:     map[string]any{"displayName": "Domain"},
		Documents:     map[string][]map[string]any{"actions": {{"identity": "legacy", "definition": map[string]any{}}}},
	}
	legacyVersion, err := ComputeForDeclaredVersion(bundle, "dv1:sha256:"+strings.Repeat("0", 64))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(legacyVersion, "dv1:sha256:") {
		t.Fatalf("legacy validation returned wrong contract: %s", legacyVersion)
	}
	bundle.Documents["actions"][0]["displayName"] = "Changed"
	changed, err := ComputeForDeclaredVersion(bundle, legacyVersion)
	if err != nil {
		t.Fatal(err)
	}
	if changed == legacyVersion {
		t.Fatal("dv1 validation did not detect changed portable content")
	}
	if _, err = ComputeForDeclaredVersion(bundle, "dv3:sha256:"+strings.Repeat("0", 64)); err == nil {
		t.Fatal("unsupported domain version was accepted")
	}
	if _, err = ComputeForDeclaredVersion(bundle, "dv1:sha256:bad"); err == nil {
		t.Fatal("malformed domain version was accepted")
	}
}

func TestComputeChangesWhenPortableContentChanges(t *testing.T) {
	bundle := entities.PortableBundle{
		Kind: "workspace-snapshot", SchemaVersion: 1,
		Workspace: map[string]any{"displayName": "Domain"},
		Documents: map[string][]map[string]any{"queries": {{"identity": "query", "source": "return 1"}}},
	}
	before, err := Compute(bundle)
	if err != nil {
		t.Fatal(err)
	}
	bundle.Documents["queries"][0]["source"] = "return 2"
	after, err := Compute(bundle)
	if err != nil {
		t.Fatal(err)
	}
	if before == after {
		t.Fatalf("portable content change must change domain version: %s", before)
	}
}

func TestComputeIgnoresWorkspaceFieldsThatImportDoesNotApply(t *testing.T) {
	left := entities.PortableBundle{
		Kind: "workspace-snapshot", SchemaVersion: 1,
		Workspace: map[string]any{
			"identity": "source", "displayName": "Domain", "dataMode": "development",
			"managedBy": "user", "managedById": "source-owner",
		},
		Documents: map[string][]map[string]any{},
	}
	right := left
	right.Workspace = map[string]any{
		"identity": "target", "displayName": "Domain", "dataMode": "development",
	}

	leftVersion, err := Compute(left)
	if err != nil {
		t.Fatal(err)
	}
	rightVersion, err := Compute(right)
	if err != nil {
		t.Fatal(err)
	}
	if leftVersion != rightVersion {
		t.Fatalf("workspace ownership fields must not affect portable version: %s != %s", leftVersion, rightVersion)
	}
}

func TestComputeCoversConfigurationDocumentsAndWorkspaceValues(t *testing.T) {
	bundle := entities.PortableBundle{
		Kind: "workspace-snapshot", SchemaVersion: 1,
		Workspace: map[string]any{"displayName": "Domain", "configuration": map[string]any{"values": map[string]any{"groundHandling": map[string]any{"rowHeight": 32}}}},
		Documents: map[string][]map[string]any{"configurations": {{"identity": "groundHandling", "sourceVersion": 1, "source": "defineConfig({ rowHeight: value(Number, 32) })", "active": true}}},
	}
	base, err := Compute(bundle)
	if err != nil {
		t.Fatal(err)
	}

	changedValue := bundle
	changedValue.Workspace = map[string]any{"displayName": "Domain", "configuration": map[string]any{"values": map[string]any{"groundHandling": map[string]any{"rowHeight": 40}}}}
	valueVersion, err := Compute(changedValue)
	if err != nil {
		t.Fatal(err)
	}
	if valueVersion == base {
		t.Fatal("workspace configuration.values did not change domain version")
	}

	changedSource := bundle
	changedSource.Documents = map[string][]map[string]any{"configurations": {{"identity": "groundHandling", "sourceVersion": 1, "source": "defineConfig({ rowHeight: value(Number, 40) })", "active": true}}}
	sourceVersion, err := Compute(changedSource)
	if err != nil {
		t.Fatal(err)
	}
	if sourceVersion == base {
		t.Fatal("Configuration source did not change domain version")
	}

	deleted := bundle
	deleted.Documents = map[string][]map[string]any{"configurations": {{"identity": "groundHandling", "sourceVersion": 1, "source": "defineConfig({ rowHeight: value(Number, 32) })", "active": true, "deletedAt": "2026-08-20T00:00:00Z"}}}
	deletedVersion, err := Compute(deleted)
	if err != nil {
		t.Fatal(err)
	}
	if deletedVersion == base {
		t.Fatal("Configuration soft-delete did not change domain version")
	}
}
