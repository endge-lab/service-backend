package workspace_state

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

func TestPortableBundleNeverCarriesBackendConnections(t *testing.T) {
	raw := []byte(`{"kind":"workspace-snapshot","schemaVersion":1,"workspace":{},"documents":{},"installedIntegrations":[],"backendConnections":[{"baseUrl":"https://must-not-transfer.example.com"}]}`)
	var bundle entities.PortableBundle
	if err := json.Unmarshal(raw, &bundle); err != nil {
		t.Fatal(err)
	}
	normalizePortableBundle(&bundle)
	exported, err := json.Marshal(bundle)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(exported), "backendConnections") || strings.Contains(string(exported), "must-not-transfer") {
		t.Fatalf("portable bundle leaked backend catalog: %s", exported)
	}
}
