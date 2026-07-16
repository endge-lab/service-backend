package converters

import "github.com/endge-lab/service-backend/internal/domain/entities"

type ConverterWithFolder struct {
	Converter      *entities.RConverter
	FolderIdentity string
}
