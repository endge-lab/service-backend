package modules

import (
	"context"
	"errors"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
)

type UseCase struct {
	repository ports.ModuleRepository
	tx         ports.TxManager
}

func NewUseCase(repository ports.ModuleRepository, tx ports.TxManager) *UseCase {
	return &UseCase{repository: repository, tx: tx}
}

func requirePublisher(ctx context.Context) (string, error) {
	actor, err := shared.Actor(ctx)
	if err != nil {
		return "", err
	}
	_, token := entities.AccessTokenFromContext(ctx)
	if !actor.PlatformAdmin || token {
		return "", domainerrors.Forbidden("module_platform_admin_required", "Publishing modules requires a platform admin session")
	}
	return actor.User.ID, nil
}

func (*UseCase) AuthorizePublish(ctx context.Context) error {
	_, err := requirePublisher(ctx)
	return err
}

func catalogError(err error) error {
	switch {
	case errors.Is(err, ports.ErrNotFound):
		return domainerrors.NotFound("module_not_found", "Module or version not found")
	case errors.Is(err, ports.ErrModuleConflict):
		return domainerrors.Conflict("module_conflict", "Module identity, revision or version conflicts with the catalog")
	default:
		return err
	}
}

func validate(value *entities.Module) error {
	value.Name = strings.TrimSpace(value.Name)
	if (value.Identity != "" && !validIdentity(value.Identity)) ||
		value.Name == "" || utf8.RuneCountInString(value.Name) > 160 || utf8.RuneCountInString(value.Description) > 4000 {
		return domainerrors.InvalidInput("module_metadata_invalid", "Invalid module identity, name or description")
	}
	return nil
}

func (u *UseCase) List(ctx context.Context, deleted bool) ([]entities.Module, error) {
	if deleted {
		if _, err := requirePublisher(ctx); err != nil {
			return nil, err
		}
	} else if _, err := shared.Actor(ctx); err != nil {
		return nil, err
	}
	return u.repository.ListModules(ctx, deleted)
}

func (u *UseCase) activeModule(ctx context.Context, identity string) (entities.Module, error) {
	value, err := u.repository.GetModule(ctx, identity)
	if err == nil && value.DeletedAt != nil {
		return entities.Module{}, ports.ErrNotFound
	}
	return value, err
}

func (u *UseCase) Detail(ctx context.Context, identity string) (entities.ModuleDetail, error) {
	if _, err := shared.Actor(ctx); err != nil {
		return entities.ModuleDetail{}, err
	}
	value, err := u.activeModule(ctx, identity)
	if err != nil {
		return entities.ModuleDetail{}, catalogError(err)
	}
	versions, err := u.repository.ListModuleVersions(ctx, value.ID)
	return entities.ModuleDetail{Module: value, Versions: versions}, catalogError(err)
}

func (u *UseCase) Create(ctx context.Context, value entities.Module, data []byte) (result entities.ModuleDetail, err error) {
	actor, err := requirePublisher(ctx)
	if err != nil {
		return result, err
	}
	if err = validate(&value); err != nil {
		return result, err
	}
	version, err := parseArchive(data, value.Identity, actor)
	if err != nil {
		return result, domainerrors.InvalidInput("module_archive_invalid", err.Error())
	}
	value.Identity = version.Manifest.Identity
	value.UpdatedBy = actor
	err = u.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		module, err := u.repository.CreateOrRestoreModule(ctx, value)
		if err != nil {
			return err
		}
		if _, err = u.repository.PublishModuleVersion(ctx, module.ID, version); err != nil {
			return err
		}
		versions, err := u.repository.ListModuleVersions(ctx, module.ID)
		if err != nil {
			return err
		}
		result = entities.ModuleDetail{Module: module, Versions: versions}
		return nil
	})
	return result, catalogError(err)
}

func (u *UseCase) Patch(ctx context.Context, identity string, revision int64, name, description *string) (entities.Module, error) {
	actor, err := requirePublisher(ctx)
	if err != nil {
		return entities.Module{}, err
	}
	value, err := u.activeModule(ctx, identity)
	if err != nil {
		return value, catalogError(err)
	}
	if name != nil {
		value.Name = *name
	}
	if description != nil {
		value.Description = *description
	}
	if err = validate(&value); err != nil {
		return value, err
	}
	value.Revision, value.UpdatedBy = revision, actor
	value, err = u.repository.PatchModule(ctx, value)
	return value, catalogError(err)
}

func (u *UseCase) SetDeleted(ctx context.Context, identity string, revision int64, deleted bool) (entities.Module, error) {
	actor, err := requirePublisher(ctx)
	if err != nil {
		return entities.Module{}, err
	}
	value, err := u.repository.GetModule(ctx, identity)
	if err != nil {
		return entities.Module{}, catalogError(err)
	}
	value, err = u.repository.SetModuleDeleted(ctx, value.ID, revision, actor, deleted)
	return value, catalogError(err)
}

func (u *UseCase) Publish(ctx context.Context, identity string, data []byte) (entities.ModuleVersion, error) {
	actor, err := requirePublisher(ctx)
	if err != nil {
		return entities.ModuleVersion{}, err
	}
	value, err := u.activeModule(ctx, identity)
	if err != nil {
		return entities.ModuleVersion{}, catalogError(err)
	}
	version, err := parseArchive(data, identity, actor)
	if err != nil {
		return version, domainerrors.InvalidInput("module_archive_invalid", err.Error())
	}
	var published entities.ModuleVersion
	err = u.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		var publishErr error
		published, publishErr = u.repository.PublishModuleVersion(ctx, value.ID, version)
		return publishErr
	})
	return published, catalogError(err)
}

func (u *UseCase) Version(ctx context.Context, identity, version string, archive bool) (entities.ModuleVersion, error) {
	if _, err := shared.Actor(ctx); err != nil {
		return entities.ModuleVersion{}, err
	}
	value, err := u.repository.GetModule(ctx, identity)
	if err != nil {
		return entities.ModuleVersion{}, catalogError(err)
	}
	result, err := u.repository.GetModuleVersion(ctx, value.ID, version, archive)
	result.ModuleArchived = value.DeletedAt != nil
	return result, catalogError(err)
}

func (*UseCase) ReadAsset(archive []byte, name string) ([]byte, error) {
	files, err := openArchive(archive)
	if err != nil {
		return nil, err
	}
	if file := files[name]; file != nil && (name == "module.js" || name == "module.css") {
		reader, err := file.Open()
		if err != nil {
			return nil, err
		}
		defer reader.Close()
		return io.ReadAll(reader)
	}
	return nil, domainerrors.NotFound("module_asset_not_found", "Module asset not found")
}
