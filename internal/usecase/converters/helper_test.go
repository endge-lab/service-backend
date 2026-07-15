package converters

import (
	"testing"

	"github.com/endge-lab/service-backend/internal/usecase/adapters"
)

func TestNormalizeAndValidateCreateInput(t *testing.T) {
	input := adapters.CreateConverterInput{ProjectIdentity: " demo ", FolderIdentity: " root-converters ", Identity: " date-format ", DisplayName: " Date format ", ConverterType: " format "}
	if err := normalizeAndValidateCreateInput(&input); err != nil {
		t.Fatal(err)
	}
	if input.ProjectIdentity != "demo" || input.FolderIdentity != "root-converters" || input.Identity != "date-format" || input.ConverterType != "format" {
		t.Fatal("input was not normalized")
	}
}
func TestNormalizeAndValidateUpdateInputRejectsMissingType(t *testing.T) {
	input := adapters.UpdateConverterInput{ProjectIdentity: "demo", FolderIdentity: "root-converters", ConverterIdentity: "date-format", DisplayName: "Date format"}
	if err := normalizeAndValidateUpdateInput(&input); err == nil {
		t.Fatal("expected validation error")
	}
}
