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
