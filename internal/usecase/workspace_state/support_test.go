package workspace_state

import (
	"context"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

func TestConfigurationFolderResolutionReturnsNil(t *testing.T) {
	coordinator := &Coordinator{}
	scope := entities.WorkspaceAccess{}

	folderID, err := coordinator.resolveFolder(context.Background(), scope, "configurations", map[string]any{})
	if err != nil || folderID != nil {
		t.Fatalf("configuration input resolved to folder: id=%v err=%v", folderID, err)
	}

	folderID, err = coordinator.resolveDocumentFolder(context.Background(), scope, entities.Document{Type: "configurations"})
	if err != nil || folderID != nil {
		t.Fatalf("configuration document resolved to folder: id=%v err=%v", folderID, err)
	}
}

func TestVocabRejectsLegacyManualAuthMode(t *testing.T) {
	input := map[string]any{
		"identity":    "airlines",
		"displayName": "Airlines",
		"authMode":    "manual",
	}
	if err := validateDocument("vocabs", input); err == nil {
		t.Fatal("legacy manual vocab auth mode was accepted")
	}
}

func TestAuthProfileAllowsPluginAdapterAndRejectsSecretConfig(t *testing.T) {
	input := map[string]any{
		"identity":    "plugin-auth",
		"displayName": "Plugin auth",
		"adapterId":   "company-plugin",
		"credentials": map[string]any{"secret": "{COMPANY_SECRET}"},
	}
	if err := validateDocument("auth-profiles", input); err != nil {
		t.Fatalf("plugin adapter rejected: %v", err)
	}
	if err := validateSecrets(map[string]any{"config": map[string]any{"password": "raw"}}); err == nil {
		t.Fatal("secret AuthProfile config was accepted")
	}
}
