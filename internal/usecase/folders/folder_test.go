package folders

import (
	"context"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	"github.com/endge-lab/service-backend/internal/usecase/adapters"
	"github.com/google/uuid"
)

func TestNormalizeAndValidateCreateInput(t *testing.T) {
	parentIdentity := " root-components "
	input := adapters.CreateFolderInput{
		ProjectIdentity: " demo-project ",
		EntityType:      entities.FolderEntityTypeComponents,
		Identity:        " shared-components ",
		DisplayName:     " Shared Components ",
		ParentIdentity:  &parentIdentity,
	}

	if err := normalizeAndValidateCreateInput(&input); err != nil {
		t.Fatalf("normalizeAndValidateCreateInput() error = %v", err)
	}
	if input.ProjectIdentity != "demo-project" {
		t.Fatalf("project identity = %q", input.ProjectIdentity)
	}
	if input.Identity != "shared-components" {
		t.Fatalf("folder identity = %q", input.Identity)
	}
	if input.DisplayName != "Shared Components" {
		t.Fatalf("display name = %q", input.DisplayName)
	}
	if input.ParentIdentity == nil || *input.ParentIdentity != "root-components" {
		t.Fatalf("parent identity = %v", input.ParentIdentity)
	}
	if input.Meta == nil {
		t.Fatal("meta must be initialized")
	}
}

func TestNormalizeAndValidateCreateInputRejectsUnsupportedEntityType(t *testing.T) {
	input := adapters.CreateFolderInput{
		ProjectIdentity: "demo-project",
		EntityType:      entities.FolderEntityType("unsupported"),
		Identity:        "folder",
		DisplayName:     "Folder",
	}

	if err := normalizeAndValidateCreateInput(&input); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateNoCycle(t *testing.T) {
	folderID := uuid.New()
	childID := uuid.New()
	grandchildID := uuid.New()

	repository := &foldersRepositoryStub{
		folders: map[uuid.UUID]*entities.Folder{
			childID: {
				ID:       childID,
				ParentID: uuidPointer(grandchildID),
			},
			grandchildID: {
				ID:       grandchildID,
				ParentID: uuidPointer(folderID),
			},
		},
	}
	service := &Folder{folderRepository: repository}

	if err := service.validateNoCycle(context.Background(), folderID, uuidPointer(childID)); err == nil {
		t.Fatal("expected folder cycle error")
	}
}

func TestValidateNoCycleAllowsUnrelatedParent(t *testing.T) {
	folderID := uuid.New()
	parentID := uuid.New()

	repository := &foldersRepositoryStub{
		folders: map[uuid.UUID]*entities.Folder{
			parentID: {ID: parentID},
		},
	}
	service := &Folder{folderRepository: repository}

	if err := service.validateNoCycle(context.Background(), folderID, uuidPointer(parentID)); err != nil {
		t.Fatalf("validateNoCycle() error = %v", err)
	}
}
