package converters

import (
	"context"
	stderrors "errors"
	"testing"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	apperrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/adapters"
	"github.com/google/uuid"
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

func TestResolveFolderID(t *testing.T) {
	projectID := uuid.New()
	folderID := uuid.New()
	identity := "root-converters"

	t.Run("returns folder ID", func(t *testing.T) {
		repository := &foldersRepositoryStub{folder: &entities.Folder{ID: folderID}}
		service := &Converter{folderRepository: repository}

		result, err := service.resolveFolderID(context.Background(), projectID, &identity)
		if err != nil {
			t.Fatal(err)
		}
		if result == nil || *result != folderID {
			t.Fatalf("folder ID = %v, want %v", result, folderID)
		}
		if repository.entityType != entities.FolderEntityTypeConverters {
			t.Fatalf("entity type = %q, want %q", repository.entityType, entities.FolderEntityTypeConverters)
		}
	})

	t.Run("maps not found to folder entity type mismatch", func(t *testing.T) {
		service := &Converter{folderRepository: &foldersRepositoryStub{
			err: apperrors.NotFound("folder_not_found", "folder not found"),
		}}

		_, err := service.resolveFolderID(context.Background(), projectID, &identity)
		if got := apperrors.CodeOf(err); got != "folder_entity_type_mismatch" {
			t.Fatalf("error code = %q, want %q", got, "folder_entity_type_mismatch")
		}
	})

	t.Run("preserves repository error", func(t *testing.T) {
		repositoryErr := stderrors.New("database unavailable")
		service := &Converter{folderRepository: &foldersRepositoryStub{err: repositoryErr}}

		_, err := service.resolveFolderID(context.Background(), projectID, &identity)
		if !stderrors.Is(err, repositoryErr) {
			t.Fatalf("error = %v, want original repository error", err)
		}
	})
}

func TestConverterWithFolders(t *testing.T) {
	firstFolderID := uuid.New()
	secondFolderID := uuid.New()

	result, err := converterWithFolders(
		[]*entities.Converter{
			{ID: uuid.New(), FolderID: secondFolderID},
			{ID: uuid.New(), FolderID: firstFolderID},
		},
		[]*entities.Folder{
			{ID: firstFolderID, Identity: "root-converters"},
			{ID: secondFolderID, Identity: "formatters"},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 {
		t.Fatalf("result length = %d, want 2", len(result))
	}
	if result[0].FolderIdentity != "formatters" || result[1].FolderIdentity != "root-converters" {
		t.Fatalf("folder identities = %q, %q", result[0].FolderIdentity, result[1].FolderIdentity)
	}
}

func TestConverterWithFoldersRejectsUnavailableFolder(t *testing.T) {
	_, err := converterWithFolders(
		[]*entities.Converter{{FolderID: uuid.New()}},
		nil,
	)
	if got := apperrors.CodeOf(err); got != "converter_folder_not_found" {
		t.Fatalf("error code = %q, want %q", got, "converter_folder_not_found")
	}
}

type foldersRepositoryStub struct {
	folder         *entities.Folder
	err            error
	getByIDFolder  *entities.Folder
	getByIDErr     error
	folders        []*entities.Folder
	listErr        error
	getByIDCalls   int
	listCalls      int
	entityType     entities.FolderEntityType
	listEntityType entities.FolderEntityType
}

func (s *foldersRepositoryStub) Create(context.Context, *entities.Folder) (*entities.Folder, error) {
	return nil, nil
}

func (s *foldersRepositoryStub) Update(context.Context, *entities.Folder) (*entities.Folder, error) {
	return nil, nil
}

func (s *foldersRepositoryStub) GetByID(context.Context, uuid.UUID) (*entities.Folder, error) {
	s.getByIDCalls++
	return s.getByIDFolder, s.getByIDErr
}

func (s *foldersRepositoryStub) GetByIDIncludingDeleted(context.Context, uuid.UUID) (*entities.Folder, error) {
	return nil, nil
}

func (s *foldersRepositoryStub) GetByIdentity(
	_ context.Context,
	_ *uuid.UUID,
	entityType entities.FolderEntityType,
	_ string,
) (*entities.Folder, error) {
	s.entityType = entityType
	return s.folder, s.err
}

func (s *foldersRepositoryStub) GetByIdentityIncludingDeleted(
	context.Context,
	*uuid.UUID,
	entities.FolderEntityType,
	string,
) (*entities.Folder, error) {
	return nil, nil
}

func (s *foldersRepositoryStub) List(
	_ context.Context,
	_ *uuid.UUID,
	entityType entities.FolderEntityType,
) ([]*entities.Folder, error) {
	s.listCalls++
	s.listEntityType = entityType
	return s.folders, s.listErr
}

func (s *foldersRepositoryStub) SoftDelete(context.Context, uuid.UUID) error { return nil }

func (s *foldersRepositoryStub) Restore(context.Context, uuid.UUID) error { return nil }

func (s *foldersRepositoryStub) HardDelete(context.Context, uuid.UUID) error { return nil }

func (s *foldersRepositoryStub) ExistsByIdentity(
	context.Context,
	*uuid.UUID,
	entities.FolderEntityType,
	string,
) (bool, error) {
	return false, nil
}

func (s *foldersRepositoryStub) Count(
	context.Context,
	*uuid.UUID,
	entities.FolderEntityType,
) (int64, error) {
	return 0, nil
}
