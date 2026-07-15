package components

import (
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/adapters"
)

func TestNormalizeAndValidateCreateInput(t *testing.T) {
	input := adapters.CreateComponentInput{ProjectIdentity: " demo ", FolderIdentity: " root-components ", Identity: " card ", DisplayName: " Card ", ComponentType: entities.ComponentTypeSFC, Source: " <template /> "}
	if err := normalizeAndValidateCreateInput(&input); err != nil {
		t.Fatal(err)
	}
	if input.ProjectIdentity != "demo" || input.FolderIdentity != "root-components" || input.Identity != "card" || input.Source != "<template />" {
		t.Fatal("input was not normalized")
	}
}
func TestNormalizeAndValidateUpdateInputRejectsUnsupportedType(t *testing.T) {
	input := adapters.UpdateComponentInput{ProjectIdentity: "demo", FolderIdentity: "root-components", ComponentIdentity: "card", DisplayName: "Card", ComponentType: "unknown", Source: "<template />"}
	if err := normalizeAndValidateUpdateInput(&input); err == nil {
		t.Fatal("expected validation error")
	}
}
