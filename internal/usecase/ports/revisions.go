package ports

import (
	"context"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

// RevisionRepository задаёт порт хранения ревизий для use case-слоя.
type RevisionRepository interface {
	NextWorkspaceSequence(context.Context, string) (int64, error)
	CreateMutationBatch(context.Context, *string, string, string) (string, error)
	LatestRevision(context.Context, *string, string, string) (*entities.Revision, error)
	InsertRevision(context.Context, entities.Revision) (*entities.Revision, error)
	ListRevisions(context.Context, string, string, string) ([]entities.Revision, error)
	GetRevision(context.Context, string, string, string, string) (*entities.Revision, error)
}
