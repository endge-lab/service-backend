package workspace_state

import (
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/domainversion"
	"github.com/endge-lab/service-backend/internal/domain/entities"
)

func TestFinalizeImportDomainVersionMigratesSFCEditingDefaultsAfterSourceVerification(t *testing.T) {
	bundle := entities.PortableBundle{
		Kind:          "workspace-snapshot",
		SchemaVersion: SchemaVersion,
		Workspace: map[string]any{
			"displayName":   "Imported workspace",
			"dataMode":      "development",
			"configuration": map[string]any{},
		},
		Documents: map[string][]map[string]any{},
	}
	sourceDomainVersion, err := domainversion.Compute(bundle)
	if err != nil {
		t.Fatalf("compute source domain version: %v", err)
	}
	bundle.DomainVersion = sourceDomainVersion

	defaultsAdded, err := finalizeImportDomainVersion(&bundle, bundle.DomainVersion)
	if err != nil {
		t.Fatalf("finalize import domain version: %v", err)
	}
	if !defaultsAdded {
		t.Fatal("SFC editing defaults were not added")
	}
	if configurations, exists := bundle.Documents["configurations"]; !exists || len(configurations) != 0 {
		t.Fatalf("legacy snapshot was not normalized with an empty configurations collection: %#v", bundle.Documents)
	}
	if bundle.DomainVersion == sourceDomainVersion {
		t.Fatal("effective domain version was not updated after migration")
	}

	effectiveDomainVersion, err := domainversion.Compute(bundle)
	if err != nil {
		t.Fatalf("compute effective domain version: %v", err)
	}
	if bundle.DomainVersion != effectiveDomainVersion {
		t.Fatalf("stored domain version does not match migrated snapshot: got %q, want %q", bundle.DomainVersion, effectiveDomainVersion)
	}
}

func TestSourceDomainVersionIsVerifiedBeforeLegacyActionMigration(t *testing.T) {
	bundle := entities.PortableBundle{
		Kind:          "workspace-snapshot",
		SchemaVersion: SchemaVersion,
		Workspace: map[string]any{
			"displayName": "Imported workspace",
			"dataMode":    "development",
		},
		Documents: map[string][]map[string]any{
			"actions": {{
				"identity":    "orders.open",
				"displayName": "Open order",
				"definition":  map[string]any{"nodes": []any{}, "edges": []any{}},
			}},
			"configurations": {},
		},
	}
	providedDomainVersion, err := domainversion.Compute(bundle)
	if err != nil {
		t.Fatalf("compute source domain version: %v", err)
	}
	bundle.DomainVersion = providedDomainVersion

	computedSourceDomainVersion, normalization, err := normalizePortableBundleForImport(&bundle)
	if err != nil {
		t.Fatalf("normalize portable bundle for import: %v", err)
	}
	if computedSourceDomainVersion != providedDomainVersion {
		t.Fatalf("source domain version mismatch: got %q, want %q", computedSourceDomainVersion, providedDomainVersion)
	}
	if normalization.MigratedLegacyActions != 1 {
		t.Fatalf("legacy Action was not migrated: %+v", normalization)
	}
	normalizedDomainVersion, err := domainversion.Compute(bundle)
	if err != nil {
		t.Fatalf("compute normalized domain version: %v", err)
	}
	if normalizedDomainVersion == providedDomainVersion {
		t.Fatal("legacy Action migration did not change domain version")
	}

	if _, err = finalizeImportDomainVersion(&bundle, providedDomainVersion); err != nil {
		t.Fatalf("finalize import domain version: %v", err)
	}
	if bundle.DomainVersion != normalizedDomainVersion {
		t.Fatalf("effective domain version mismatch: got %q, want %q", bundle.DomainVersion, normalizedDomainVersion)
	}
}
