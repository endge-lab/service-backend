package adapters

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

type CreateDataViewInput struct {
	ProjectIdentity string
	FolderIdentity  string
	QueryIdentity   string

	Identity    string
	DisplayName string
	Description *string

	ViewType     string
	Source       map[string]any
	InputSchema  map[string]any
	OutputSchema map[string]any
	Meta         map[string]any
	Active       bool
}

type UpdateDataViewInput struct {
	ProjectIdentity  string
	DataViewIdentity string
	FolderIdentity   string
	QueryIdentity    string

	DisplayName string
	Description *string

	ViewType     string
	Source       map[string]any
	InputSchema  map[string]any
	OutputSchema map[string]any
	Meta         map[string]any
	Active       bool
}

type GetDataViewInput struct {
	ProjectIdentity  string
	DataViewIdentity string
}

type DataViewIdentityInput struct {
	ProjectIdentity  string
	DataViewIdentity string
}

type ListDataViewsInput struct {
	ProjectIdentity string
	FolderIdentity  *string
	QueryIdentity   *string
}

type DataViewWithRelations struct {
	DataView       *entities.DataView
	FolderIdentity string
	QueryIdentity  string
}

type DataViewService interface {
	Create(ctx context.Context, input CreateDataViewInput) (*DataViewWithRelations, error)
	Update(ctx context.Context, input UpdateDataViewInput) (*DataViewWithRelations, error)
	GetByIdentity(ctx context.Context, input GetDataViewInput) (*DataViewWithRelations, error)
	List(ctx context.Context, input ListDataViewsInput) ([]*DataViewWithRelations, error)
	SoftDelete(ctx context.Context, input DataViewIdentityInput) error
	Restore(ctx context.Context, input DataViewIdentityInput) error
	HardDelete(ctx context.Context, input DataViewIdentityInput) error
	Count(ctx context.Context, input ListDataViewsInput) (int64, error)
}
