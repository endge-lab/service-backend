package workspace_state

import (
	"encoding/json"
	"strings"
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

func TestNormalizePortableBundleDoesNotTreatSoftDeletedAsFolder(t *testing.T) {
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
	if result.IgnoredLegacyFolders != 3 || len(bundle.Documents["folders"]) != 1 {
		t.Fatalf("legacy folders were not ignored: %#v", result)
	}
	if result.NormalizedFolderReferences != 4 {
		t.Fatalf("legacy folder references were not normalized: %#v", result)
	}
	if len(bundle.Documents["types"]) != 2 || stringField(bundle.Documents["types"][0], "folderIdentity") != "root-types" {
		t.Fatalf("soft-deleted must be handled as an ordinary mismatched folder: %#v", bundle.Documents["types"])
	}
	if _, exists := bundle.Documents["types"][1]["folderIdentity"]; exists {
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

func TestNormalizePortableBundleMigratesLegacyExternalPayloadVocab(t *testing.T) {
	bundle := entities.PortableBundle{Documents: map[string][]map[string]any{
		"vocabs": {{
			"identity": "airlines", "displayName": "Airlines",
			"mode": "external_payload", "baseApiUrl": "{ENDPOINT_VOCABS_SERVICE}",
			"collectionSlug": "airlines", "authMode": "inherit",
		}},
	}}

	result := normalizePortableBundle(&bundle)
	vocab := bundle.Documents["vocabs"][0]
	if result.MigratedLegacyVocabs != 1 {
		t.Fatalf("legacy Vocab was not migrated: %+v", result)
	}
	if version, ok := numberField(vocab, "sourceVersion"); !ok || version != 1 {
		t.Fatalf("sourceVersion was not set: %#v", vocab)
	}
	for _, fragment := range []string{`provider: payload({`, `baseUrl: env("ENDPOINT_VOCABS_SERVICE")`, `collection: "airlines"`, `auth: { mode: "inherit" }`} {
		if !strings.Contains(stringField(vocab, "source"), fragment) {
			t.Fatalf("migrated source does not contain %q: %s", fragment, stringField(vocab, "source"))
		}
	}
}
