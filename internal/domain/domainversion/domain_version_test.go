package domainversion

import (
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
