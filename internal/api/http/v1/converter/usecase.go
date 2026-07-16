package converter

import (
	"context"

	"github.com/endge-lab/service-backend/internal/usecase/converters"
)

// UseCase is the application contract consumed by the converter HTTP adapter.
type UseCase interface {
	Create(ctx context.Context, input converters.CreateConverterInput) (*converters.ConverterWithFolder, error)
	Update(ctx context.Context, input converters.UpdateConverterInput) (*converters.ConverterWithFolder, error)
	GetByIdentity(ctx context.Context, input converters.GetConverterInput) (*converters.ConverterWithFolder, error)
	List(ctx context.Context, input converters.ListConvertersInput) ([]*converters.ConverterWithFolder, error)
	SoftDelete(ctx context.Context, input converters.ConverterIdentityInput) error
	Restore(ctx context.Context, input converters.ConverterIdentityInput) error
	HardDelete(ctx context.Context, input converters.ConverterIdentityInput) error
}
