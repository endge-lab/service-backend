package workspace_state

import (
	"encoding/json"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

// FuzzPortableBundleStructure проверяет нормализацию и relation-анализ произвольного snapshot JSON.
func FuzzPortableBundleStructure(f *testing.F) {
	f.Add(`{"kind":"workspace-snapshot","schemaVersion":1,"workspace":{},"documents":{},"installedIntegrations":[]}`)
	f.Add(`{"documents":{"folders":[{"identity":"a","parentIdentity":"b"}]}}`)
	f.Add(`null`)
	f.Fuzz(func(t *testing.T, raw string) {
		if len(raw) > 256*1024 {
			t.Skip()
		}
		var bundle entities.PortableBundle
		if json.Unmarshal([]byte(raw), &bundle) != nil {
			return
		}
		normalizePortableBundle(&bundle)
		_ = validateSnapshotRelations(bundle)
		for kind, items := range bundle.Documents {
			_, _ = orderPortableItems(kind, items)
		}
	})
}

func TestNormalizePortableBundleIgnoresLegacyImportState(t *testing.T) {
	bundle := entities.PortableBundle{
		Workspace:             map[string]any{},
		InstalledIntegrations: []map[string]any{{"identity": "legacy"}},
		Documents: map[string][]map[string]any{
			"folders": {
				{"identity": "soft-deleted", "entityType": "system"},
				{"identity": "no-folder", "entityType": "system"},
				{"identity": "root-bindings", "entityType": "behavior-bindings"},
				{"identity": "query-folder", "entityType": "queries"},
			},
			"types": {
				{"identity": "deleted", "folderIdentity": "soft-deleted"},
				{"identity": "active", "folderIdentity": "no-folder"},
			},
			"streams": {
				{"identity": "events", "folderIdentity": "root-streams"},
			},
			"compositions": {
				{"identity": "composition", "folderIdentity": "query-folder"},
			},
		},
	}

	result := normalizePortableBundle(&bundle)

	if result.IgnoredIntegrations != 1 || len(bundle.InstalledIntegrations) != 0 {
		t.Fatalf("installed integrations were not ignored: %#v", result)
	}
	if result.IgnoredDeletedDocuments != 1 || len(bundle.Documents["types"]) != 1 {
		t.Fatalf("soft-deleted documents were not ignored: %#v", result)
	}
	if result.IgnoredLegacyFolders != 3 || len(bundle.Documents["folders"]) != 1 {
		t.Fatalf("legacy folders were not ignored: %#v", result)
	}
	if _, exists := bundle.Documents["types"][0]["folderIdentity"]; exists {
		t.Fatal("no-folder reference was not normalized to the collection root")
	}
	if folder := stringField(bundle.Documents["streams"][0], "folderIdentity"); folder != "root-queries" {
		t.Fatalf("stream root was not normalized: %q", folder)
	}
	if folder := stringField(bundle.Documents["compositions"][0], "folderIdentity"); folder != "root-compositions" {
		t.Fatalf("cross-collection folder was not normalized: %q", folder)
	}
}

func TestNormalizePortableBundleMigratesQueryV1(t *testing.T) {
	bundle := entities.PortableBundle{Documents: map[string][]map[string]any{
		"queries": {{"identity": "query-a", "displayName": "Query A", "source": "query {}", "sourceVersion": float64(1)}},
		"pages":   {},
	}}
	normalizePortableBundle(&bundle)
	if version, ok := numberField(bundle.Documents["queries"][0], "sourceVersion"); !ok || version != 2 {
		t.Fatalf("query sourceVersion was not migrated to v2: %#v", bundle.Documents["queries"][0])
	}
}
