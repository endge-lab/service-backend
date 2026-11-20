package ports

import (
	"context"
	"errors"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

var ErrModuleConflict = errors.New("module identity, revision or version conflict")

type ModuleRepository interface {
	ListModules(context.Context, bool) ([]entities.Module, error)
	GetModule(context.Context, string) (entities.Module, error)
	CreateOrRestoreModule(context.Context, entities.Module) (entities.Module, error)
	PatchModule(context.Context, entities.Module) (entities.Module, error)
	SetModuleDeleted(context.Context, string, int64, string, bool) (entities.Module, error)
	ListModuleVersions(context.Context, string) ([]entities.ModuleVersion, error)
	GetModuleVersion(context.Context, string, string, bool) (entities.ModuleVersion, error)
	PublishModuleVersion(context.Context, string, entities.ModuleVersion) (entities.ModuleVersion, error)
}
