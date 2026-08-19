package workspace_state

import (
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/domainversion"
	"github.com/endge-lab/service-backend/internal/domain/entities"
)

func TestPrepareImportDomainVersionMigratesSFCEditingDefaultsAfterSourceVerification(t *testing.T) {
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

	computedSourceDomainVersion, defaultsAdded, err := prepareImportDomainVersion(&bundle, bundle.DomainVersion)
	if err != nil {
		t.Fatalf("prepare import domain version: %v", err)
	}
	if computedSourceDomainVersion != sourceDomainVersion {
		t.Fatalf("source domain version changed before verification: got %q, want %q", computedSourceDomainVersion, sourceDomainVersion)
	}
	if !defaultsAdded {
		t.Fatal("SFC editing defaults were not added")
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
