package workspace_state

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

func TestPortableBundleNeverCarriesAccessGrantsOrServiceUsers(t *testing.T) {
	raw := []byte(`{"kind":"workspace-snapshot","schemaVersion":1,"workspace":{},"documents":{},"installedIntegrations":[],"accessGrants":[{"role":"admin"}],"serviceUsers":[{"username":"must-not-transfer"}]}`)
	var bundle entities.PortableBundle
	if err := json.Unmarshal(raw, &bundle); err != nil {
		t.Fatal(err)
	}
	normalizePortableBundle(&bundle)
	exported, err := json.Marshal(bundle)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"accessGrants", "serviceUsers", "must-not-transfer"} {
		if strings.Contains(string(exported), forbidden) {
			t.Fatalf("portable bundle leaked platform access data %q: %s", forbidden, exported)
		}
	}
}
