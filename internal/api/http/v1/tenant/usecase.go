package tenant

import (
	"context"

	"github.com/endge-lab/service-backend/internal/usecase/tenants"
)

// UseCase is the tenant application contract consumed by the HTTP adapter.
type UseCase interface {
	Create(context.Context, tenants.CreateTenantInput) (*tenants.TenantView, error)
	List(context.Context, tenants.ListTenantsInput) ([]*tenants.TenantView, error)
	GetByIdentity(context.Context, string) (*tenants.TenantView, error)
	Update(context.Context, tenants.UpdateTenantInput) (*tenants.TenantView, error)
	HardDelete(context.Context, string) error
}
