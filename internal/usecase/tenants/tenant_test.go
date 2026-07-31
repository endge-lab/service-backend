package tenants

import (
	"context"
	"testing"
	"time"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	apperrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/observability"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

func TestCreateEnsuresTenantRootFolder(t *testing.T) {
	t.Parallel()

	tenantsRepository := &tenantsRepositoryStub{}
	foldersRepository := &foldersRepositoryStub{}
	tx := &txManagerStub{}
	service := newTenantService(tenantsRepository, foldersRepository, tx)
	workspaceID := uuid.New()

	created, err := service.Create(entities.WithWorkspaceID(context.Background(), workspaceID), CreateTenantInput{
		Identity:      "tenant-default",
		DisplayName:   "Default tenant",
		Code:          "TENANT_DEFAULT",
		Configuration: contributionPtr(entities.DefaultEndgeConfigurationContribution()),
	})
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	if !tx.called || tx.rolledBack {
		t.Fatalf("transaction state: called=%v rolledBack=%v", tx.called, tx.rolledBack)
	}
	if foldersRepository.createCalls != 1 {
		t.Fatalf("root folder create calls = %d, want 1", foldersRepository.createCalls)
	}
	if foldersRepository.created[0].Identity != entities.TenantRootFolderIdentity || !foldersRepository.created[0].IsRoot || !foldersRepository.created[0].IsSystem {
		t.Fatalf("unexpected root folder: %+v", foldersRepository.created[0])
	}
	if created.Tenant.FolderID == nil || tenantsRepository.created.FolderID == nil || *created.Tenant.FolderID != *tenantsRepository.created.FolderID || created.FolderIdentity == nil || *created.FolderIdentity != entities.TenantRootFolderIdentity {
		t.Fatalf("tenant root folder was not resolved: %+v", created)
	}
}

func TestCreateRejectsUnsupportedConfigurationPatch(t *testing.T) {
	t.Parallel()

	configuration := entities.DefaultEndgeConfigurationContribution()
	configuration.Patch["not-supported"] = []byte(`{"op":"set","value":true}`)
	service := newTenantService(&tenantsRepositoryStub{}, &foldersRepositoryStub{}, &txManagerStub{})

	_, err := service.Create(entities.WithWorkspaceID(context.Background(), uuid.New()), CreateTenantInput{
		Identity: "tenant-default", DisplayName: "Default tenant", Code: "TENANT_DEFAULT", Configuration: &configuration,
	})
	if code := apperrors.CodeOf(err); code != "configuration_invalid" {
		t.Fatalf("error code = %q, want configuration_invalid", code)
	}
}

func TestCreateDefaultsConfiguration(t *testing.T) {
	t.Parallel()

	repository := &tenantsRepositoryStub{}
	service := newTenantService(repository, &foldersRepositoryStub{}, &txManagerStub{})
	_, err := service.Create(entities.WithWorkspaceID(context.Background(), uuid.New()), CreateTenantInput{
		Identity: "tenant-default", DisplayName: "Default tenant", Code: "TENANT_DEFAULT",
	})
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	if repository.created == nil || repository.created.Configuration.Mode != entities.EndgeConfigurationContributionModeInherit || len(repository.created.Configuration.Patch) != 0 {
		t.Fatalf("default configuration = %#v, want clean inherit", repository.created)
	}
}

func TestUpdatePreservesImmutableFieldsAndHandlesExplicitFolderNull(t *testing.T) {
	t.Parallel()

	workspaceID := uuid.New()
	rootID := uuid.New()
	oldFolderID := uuid.New()
	createdAt := time.Date(2026, time.July, 25, 0, 0, 0, 0, time.UTC)
	current := &entities.RTenant{
		ID: uuid.New(), WorkspaceID: workspaceID, Identity: "tenant-default", DisplayName: "Old", Code: "OLD",
		FolderID: &oldFolderID, Configuration: entities.DefaultEndgeConfigurationContribution(), CreatedAt: createdAt,
	}
	root := &entities.RFolder{ID: rootID, WorkspaceID: workspaceID, EntityType: entities.FolderEntityTypeTenants, Identity: entities.TenantRootFolderIdentity, IsRoot: true, IsSystem: true}
	tenantsRepository := &tenantsRepositoryStub{byIdentity: current}
	foldersRepository := &foldersRepositoryStub{byIdentity: root}
	service := newTenantService(tenantsRepository, foldersRepository, &txManagerStub{})
	displayName := "New name"

	updated, err := service.Update(entities.WithWorkspaceID(context.Background(), workspaceID), UpdateTenantInput{
		Identity: "tenant-default", DisplayName: &displayName,
		Description:    NullableString{Set: true, Value: nil},
		FolderIdentity: NullableString{Set: true, Value: nil},
	})
	if err != nil {
		t.Fatalf("update tenant: %v", err)
	}
	if updated.Tenant.ID != current.ID || updated.Tenant.Identity != current.Identity || updated.Tenant.WorkspaceID != workspaceID || !updated.Tenant.CreatedAt.Equal(createdAt) {
		t.Fatalf("immutable tenant fields changed: %+v", updated)
	}
	if updated.Tenant.DisplayName != displayName || updated.Tenant.Description != nil || updated.Tenant.FolderID == nil || *updated.Tenant.FolderID != rootID || updated.FolderIdentity == nil || *updated.FolderIdentity != entities.TenantRootFolderIdentity {
		t.Fatalf("partial update was not applied: %+v", updated)
	}
	if foldersRepository.createCalls != 0 {
		t.Fatalf("existing tenant root must not be recreated: %d", foldersRepository.createCalls)
	}
}

func TestValidateConfigurationContributionRejectsRequiredScalarRemove(t *testing.T) {
	t.Parallel()

	configuration := entities.DefaultEndgeConfigurationContribution()
	configuration.Patch[entities.EndgeConfigurationPatchKeyDefaultTheme] = []byte(`{"op":"remove"}`)
	if err := validateConfigurationContribution(configuration); apperrors.CodeOf(err) != "configuration_invalid" {
		t.Fatalf("error = %v, want configuration_invalid", err)
	}
}

func TestResolveTenantFolderReportsContractErrors(t *testing.T) {
	t.Parallel()

	workspaceID := uuid.New()
	for _, tc := range []struct {
		name        string
		folder      *entities.RFolder
		wantCode    string
		lookupError error
	}{
		{name: "workspace mismatch", folder: &entities.RFolder{WorkspaceID: uuid.New(), EntityType: entities.FolderEntityTypeTenants}, wantCode: "folder_workspace_mismatch"},
		{name: "entity type mismatch", folder: &entities.RFolder{WorkspaceID: workspaceID, EntityType: entities.FolderEntityTypeQueries}, wantCode: "folder_entity_type_mismatch"},
		{name: "not found", lookupError: apperrors.NotFound("not_found", "folder not found"), wantCode: "folder_not_found"},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			service := newTenantService(&tenantsRepositoryStub{}, &foldersRepositoryStub{byIdentity: tc.folder, getErr: tc.lookupError}, &txManagerStub{})
			_, err := service.resolveTenantFolder(entities.WithWorkspaceID(context.Background(), workspaceID), workspaceID, "tenant-folder")
			if code := apperrors.CodeOf(err); code != tc.wantCode {
				t.Fatalf("error code = %q, want %q", code, tc.wantCode)
			}
		})
	}
}

func newTenantService(repository ports.TenantsRepository, folders ports.FoldersRepository, tx ports.TxManager) *Tenant {
	return NewTenantService(TenantParams{
		Repository: repository, FolderRepository: folders, TxManager: tx,
		Observability: observability.NewCore(otel.Tracer("test"), zap.NewNop()),
	})
}

func contributionPtr(value entities.EndgeConfigurationContribution) *entities.EndgeConfigurationContribution {
	return &value
}

type txManagerStub struct{ called, rolledBack bool }

func (s *txManagerStub) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	s.called = true
	if err := fn(ctx); err != nil {
		s.rolledBack = true
		return err
	}
	return nil
}

type tenantsRepositoryStub struct{ created, updated, byIdentity *entities.RTenant }

func (s *tenantsRepositoryStub) Create(_ context.Context, tenant *entities.RTenant) (*entities.RTenant, error) {
	copy := *tenant
	copy.ID = uuid.New()
	s.created = &copy
	return &copy, nil
}
func (s *tenantsRepositoryStub) List(context.Context, ports.TenantsFilter) ([]*entities.RTenant, error) {
	return nil, nil
}
func (s *tenantsRepositoryStub) GetByIdentity(context.Context, string) (*entities.RTenant, error) {
	if s.byIdentity == nil {
		return nil, apperrors.NotFound("not_found", "tenant not found")
	}
	copy := *s.byIdentity
	return &copy, nil
}
func (s *tenantsRepositoryStub) Update(_ context.Context, tenant *entities.RTenant) (*entities.RTenant, error) {
	copy := *tenant
	s.updated = &copy
	return &copy, nil
}
func (s *tenantsRepositoryStub) HardDelete(context.Context, string) error { return nil }

type foldersRepositoryStub struct {
	byIdentity  *entities.RFolder
	created     []*entities.RFolder
	createCalls int
	getErr      error
}

func (s *foldersRepositoryStub) Create(_ context.Context, folder *entities.RFolder) (*entities.RFolder, error) {
	copy := *folder
	if copy.ID == uuid.Nil {
		copy.ID = uuid.New()
	}
	s.createCalls++
	s.created = append(s.created, &copy)
	s.byIdentity = &copy
	return &copy, nil
}
func (s *foldersRepositoryStub) Update(context.Context, *entities.RFolder) (*entities.RFolder, error) {
	return nil, nil
}
func (s *foldersRepositoryStub) GetByID(context.Context, uuid.UUID) (*entities.RFolder, error) {
	if s.byIdentity == nil {
		return nil, apperrors.NotFound("not_found", "folder not found")
	}
	copy := *s.byIdentity
	return &copy, nil
}
func (s *foldersRepositoryStub) GetByIDIncludingDeleted(context.Context, uuid.UUID) (*entities.RFolder, error) {
	return nil, nil
}
func (s *foldersRepositoryStub) GetByIdentity(context.Context, *uuid.UUID, entities.FolderEntityType, string) (*entities.RFolder, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	if s.byIdentity == nil {
		return nil, apperrors.NotFound("not_found", "folder not found")
	}
	copy := *s.byIdentity
	return &copy, nil
}
func (s *foldersRepositoryStub) GetByIdentityIncludingDeleted(context.Context, *uuid.UUID, entities.FolderEntityType, string) (*entities.RFolder, error) {
	return nil, nil
}
func (s *foldersRepositoryStub) List(context.Context, *uuid.UUID, entities.FolderEntityType) ([]*entities.RFolder, error) {
	return nil, nil
}
func (s *foldersRepositoryStub) SoftDelete(context.Context, uuid.UUID) error { return nil }
func (s *foldersRepositoryStub) Restore(context.Context, uuid.UUID) error    { return nil }
func (s *foldersRepositoryStub) HardDelete(context.Context, uuid.UUID) error { return nil }
func (s *foldersRepositoryStub) ExistsByIdentity(context.Context, *uuid.UUID, entities.FolderEntityType, string) (bool, error) {
	return false, nil
}
func (s *foldersRepositoryStub) Count(context.Context, *uuid.UUID, entities.FolderEntityType) (int64, error) {
	return 0, nil
}
