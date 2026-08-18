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
