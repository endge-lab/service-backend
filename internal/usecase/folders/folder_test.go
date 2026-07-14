package folders

import (
	"context"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	apperrors "github.com/endge-lab/service-backend/internal/domain/errors"
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

func TestValidateNoCycleRejectsSelfParent(t *testing.T) {
	folderID := uuid.New()

	service := &Folder{
		folderRepository: &foldersRepositoryStub{},
	}

	err := service.validateNoCycle(
		context.Background(),
		folderID,
		uuidPointer(folderID),
	)
	if err == nil {
		t.Fatal("expected folder cycle error")
	}
}

func TestUpdateRejectsSelfParent(t *testing.T) {
	projectID := uuid.New()
	folderID := uuid.New()
	parentIdentity := "folder"

	repository := &foldersRepositoryStub{
		foldersByIdentity: map[string]*entities.Folder{
			"folder": {
				ID:         folderID,
				ProjectID:  uuidPointer(projectID),
				EntityType: entities.FolderEntityTypeComponents,
				Identity:   "folder",
			},
		},
	}

	service := &Folder{
		folderRepository: repository,
		projectRepository: &projectsRepositoryStub{
			project: &entities.Project{
				ID:       projectID,
				Identity: "demo-project",
			},
		},
	}

	_, err := service.Update(context.Background(), adapters.UpdateFolderInput{
		ProjectIdentity: "demo-project",
		EntityType:      entities.FolderEntityTypeComponents,
		Identity:        "folder",
		DisplayName:     "Folder",
		ParentIdentity:  &parentIdentity,
	})

	if err == nil {
		t.Fatal("expected folder cycle error")
	}
}

func TestValidateNoCycleRejectsExistingCycleInParentChain(t *testing.T) {
	folderID := uuid.New()
	parentA := uuid.New()
	parentB := uuid.New()

	repository := &foldersRepositoryStub{
		folders: map[uuid.UUID]*entities.Folder{
			parentA: {
				ID:       parentA,
				ParentID: uuidPointer(parentB),
			},
			parentB: {
				ID:       parentB,
				ParentID: uuidPointer(parentA),
			},
		},
	}

	service := &Folder{folderRepository: repository}

	err := service.validateNoCycle(
		context.Background(),
		folderID,
		uuidPointer(parentA),
	)
	if err == nil {
		t.Fatal("expected folder cycle error")
	}
}
func TestResolveParentIDRejectsWrongProjectOrEntityType(t *testing.T) {
	projectID := uuid.New()
	parentIdentity := "root-components"

	repository := &foldersRepositoryStub{
		getByIdentityErr: apperrors.NotFound("not_found", "folder not found"),
	}

	service := &Folder{folderRepository: repository}

	_, err := service.resolveParentID(
		context.Background(),
		projectID,
		entities.FolderEntityTypeComponents,
		&parentIdentity,
	)
	if err == nil {
		t.Fatal("expected validation error")
	}
}
