package queries

import "github.com/endge-lab/service-backend/internal/domain/entities"

type QueryWithFolder struct {
	Query          *entities.RQuery
	FolderIdentity string
}
