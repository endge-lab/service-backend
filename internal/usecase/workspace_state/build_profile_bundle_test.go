package workspace_state

import (
	"context"
	"encoding/json"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/build_profiles"
	"testing"
)

func TestBundleProfileSettingsSurvivePortableJSONAndImport(t *testing.T) {
	for _, settings := range []string{
		`{"buildScope":"complete-model","contexts":"all-contexts","diagnostics":"detailed","debuggerStructure":"complete-catalog","topology":[{"node":"frontend","runtime":"ts-browser"}],"includeAst":true,"fileFormat":"json"}`,
		`{"buildScope":"complete-model","contexts":"all-contexts","diagnostics":"detailed","debuggerStructure":"complete-catalog","topology":[{"node":"frontend","runtime":"ts-browser"}]}`,
	} {
		normalized, err := build_profiles.DecodeSettings(json.RawMessage(settings))
		if err != nil {
			t.Fatal(err)
		}
		bundle := entities.PortableBundle{BuildProfiles: []entities.PortableBuildProfile{{Identity: "71e7eeb2-473d-41fc-a641-ea2d9f69379b", DisplayName: "Build", Visibility: "shared", SettingsVersion: entities.BuildProfileSettingsVersion, Settings: normalized}}}
		bytes, err := json.Marshal(bundle)
		if err != nil {
			t.Fatal(err)
		}
		var incoming entities.PortableBundle
		if err = json.Unmarshal(bytes, &incoming); err != nil {
			t.Fatal(err)
		}
		plan := &entities.ImportPlan{Valid: true}
		if err = (&Coordinator{}).normalizeAdjunctForImport(context.Background(), &incoming, entities.CurrentActor{}, plan); err != nil {
			t.Fatal(err)
		}
		if !plan.Valid || len(incoming.BuildProfiles) != 1 {
			t.Fatalf("unexpected plan: %+v", plan)
		}
		if string(incoming.BuildProfiles[0].Settings) != string(normalized) {
			t.Fatal("bundle settings changed during transfer")
		}
	}
}
