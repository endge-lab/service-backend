package component_legacy

import (
	"context"

	usecase "github.com/endge-lab/service-backend/internal/usecase/components_legacy"
)

// UseCase is the application contract consumed by the legacy component HTTP adapter.
type UseCase interface {
	Create(ctx context.Context, input usecase.CreateComponentLegacyInput) (*usecase.ComponentLegacyWithFolder, error)
	Update(ctx context.Context, input usecase.UpdateComponentLegacyInput) (*usecase.ComponentLegacyWithFolder, error)
	GetByIdentity(ctx context.Context, input usecase.GetComponentLegacyInput) (*usecase.ComponentLegacyWithFolder, error)
	List(ctx context.Context, input usecase.ListComponentsLegacyInput) ([]*usecase.ComponentLegacyWithFolder, error)
	SoftDelete(ctx context.Context, input usecase.ComponentLegacyIdentityInput) error
	Restore(ctx context.Context, input usecase.ComponentLegacyIdentityInput) error
	HardDelete(ctx context.Context, input usecase.ComponentLegacyIdentityInput) error
}
