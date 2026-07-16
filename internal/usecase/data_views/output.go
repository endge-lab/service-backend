package data_views

import "github.com/endge-lab/service-backend/internal/domain/entities"

type DataViewWithRelations struct {
	DataView       *entities.RDataView
	FolderIdentity string
	QueryIdentity  string
}
