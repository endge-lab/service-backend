package adapters

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

type CreateConverterInput struct {
	ProjectIdentity string
	FolderIdentity  string

	Identity      string
	DisplayName   string
	Description   *string
	ConverterType string
	Source        map[string]any

	IsSystem bool
	Meta     map[string]any
	Active   bool
}

type UpdateConverterInput struct {
	ProjectIdentity   string
	ConverterIdentity string
	FolderIdentity    string

	DisplayName   string
	Description   *string
	ConverterType string
	Source        map[string]any

	IsSystem bool
	Meta     map[string]any
	Active   bool
}

type GetConverterInput struct {
	ProjectIdentity   string
	ConverterIdentity string
}

type ConverterIdentityInput struct {
	ProjectIdentity   string
	ConverterIdentity string
}

type ListConvertersInput struct {
	ProjectIdentity string
	FolderIdentity  *string
}

type ConverterWithFolder struct {
	Converter      *entities.Converter
	FolderIdentity string
}

type ConverterService interface {
	Create(ctx context.Context, input CreateConverterInput) (*ConverterWithFolder, error)

	Update(ctx context.Context, input UpdateConverterInput) (*ConverterWithFolder, error)

	GetByIdentity(ctx context.Context, input GetConverterInput) (*ConverterWithFolder, error)

	List(ctx context.Context, input ListConvertersInput) ([]*ConverterWithFolder, error)

	SoftDelete(ctx context.Context, input ConverterIdentityInput) error

	Restore(ctx context.Context, input ConverterIdentityInput) error

	HardDelete(ctx context.Context, input ConverterIdentityInput) error

	Count(ctx context.Context, input ListConvertersInput) (int64, error)
}
