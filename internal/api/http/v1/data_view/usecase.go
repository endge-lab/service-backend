package data_view

import (
	"context"

	"github.com/endge-lab/service-backend/internal/usecase/data_views"
)

// UseCase is the application contract consumed by the data view HTTP adapter.
type UseCase interface {
	Create(ctx context.Context, input data_views.CreateDataViewInput) (*data_views.DataViewWithRelations, error)
	Update(ctx context.Context, input data_views.UpdateDataViewInput) (*data_views.DataViewWithRelations, error)
	GetByIdentity(ctx context.Context, input data_views.GetDataViewInput) (*data_views.DataViewWithRelations, error)
	List(ctx context.Context, input data_views.ListDataViewsInput) ([]*data_views.DataViewWithRelations, error)
	SoftDelete(ctx context.Context, input data_views.DataViewIdentityInput) error
	Restore(ctx context.Context, input data_views.DataViewIdentityInput) error
	HardDelete(ctx context.Context, input data_views.DataViewIdentityInput) error
}
